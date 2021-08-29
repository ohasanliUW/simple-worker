package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	lib "simple-worker/job"
	"syscall"
	"time"

	pb "simple-worker/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedWorkerServer
	jobs map[string]map[int64]*lib.Job // all jobs created by server
}

// a custom error that will be returned when a request is not authorized
type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}

const (
	port = "localhost:50051"
)

func main() {
	srv := &server{
		jobs: make(map[string]map[int64]*lib.Job),
	}

	gsrv := grpc.NewServer(grpc.Creds(loadKeyPair()))

	pb.RegisterWorkerServer(gsrv, srv)

	go func() {
		lis, err := net.Listen("tcp", port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("server listening at %v", lis.Addr())
		if err := gsrv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Graceful stop via go channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	gsrv.GracefulStop()
	log.Println("Server exiting gracefully")
}

func loadKeyPair() credentials.TransportCredentials {
	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		panic("Failed to load server certificate")
	}

	ca_data, err := ioutil.ReadFile("certs/rootCA.crt")
	if err != nil {
		panic("Failed to read root CA data")
	}

	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM(ca_data) {
		panic("Failed to add CA certificate")
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    capool,
	}

	return credentials.NewTLS(tlsConfig)
}

func (s *server) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartResponse, error) {

	command := in.GetCommand()

	if command == "" {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"must provide a valid command",
		)
	}

	job, err := s.startJob("dummy", command)

	// Even though job.Start() fails, it is a valid job as it was successfully created.
	// Seems gRPC does not send back response if error is not nil.
	// Hence, we should return an error that has Job ID within its message
	if job != nil && err != nil {
		msg := fmt.Sprintf("Job ID %v: %v", job.Id(), err.Error())
		err = errors.New(msg)
	}
	return &pb.StartResponse{
		JobId: job.Id(),
	}, err
}

func (s *server) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopResponse, error) {

	job_id := in.GetJobId()
	err := s.stopJob("dummy", job_id)

	if err != nil {
		return nil, err
	}

	return &pb.StopResponse{
		Message: fmt.Sprintf("Successfully stopped job %v", job_id),
	}, nil
}

func (s *server) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusResponse, error) {
	job_id := in.GetJobId()

	status, err := s.statusOfJob("dummy", job_id)
	if err != nil {
		return nil, err
	}

	return &pb.StatusResponse{
		JobId:    job_id,
		Status:   status.Status,
		Exited:   status.Exited,
		ExitCode: int32(status.ExitCode),
	}, nil
}

func (s *server) Output(in *pb.OutputRequest, stream pb.Worker_OutputServer) (err error) {
	job_id := in.GetJobId()

	streamC, err := s.outputOfJob("dummy", job_id)
	if err != nil {
		return err
	}

	// try sending up to 10 lines at a time
	// if stream channel does not have data within 1 millisecond, send whatever we already have
	maxLinesPerRPC := 10
	maxWaitTime := time.Millisecond
	for {
		// read until channel is closed, which indicates end of the job
		batch, end := getNextLineBatch(streamC, maxLinesPerRPC, maxWaitTime)
		// if batch has length of 0, but stream has not ended, then try again
		if len(batch) == 0 && !end {
			continue
		}
		resp := &pb.OutputResponse{
			OutputLine: batch,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		if end {
			break
		}
	}

	return nil
}

// createJob creates a job with command for user with username
func (s *server) startJob(username string, command string) (*lib.Job, error) {
	job := lib.NewJob(command)

	if job == nil {
		return nil, errors.New("failed to create a job")
	}
	userJobs, exists := s.jobs[username]
	if !exists {
		s.jobs[username] = make(map[int64]*lib.Job)
		userJobs = s.jobs[username]
	}
	userJobs[job.Id()] = job

	if err := job.Start(); err != nil {
		return job, err
	}

	return job, nil
}

func (s *server) stopJob(username string, job_id int64) error {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return authErr
	}

	job := s.jobs[username][job_id]
	err := job.Stop()
	return err
}

func (s *server) statusOfJob(username string, job_id int64) (lib.JobStatus, error) {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return lib.JobStatus{}, authErr
	}

	job := s.jobs[username][job_id]
	status := job.Status()
	return status, nil
}

func (s *server) outputOfJob(username string, job_id int64) (chan string, error) {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return nil, authErr
	}

	job := s.jobs[username][job_id]
	outStreamC := job.Output()

	if outStreamC == nil {
		return nil, errors.New("could not get output of the job")
	}

	return outStreamC, nil
}

func getNextLineBatch(streamC chan string, size int, d time.Duration) ([]string, bool) {
	batch := make([]string, 0, size)
	count := 0

	for {
		select {
		case <-time.After(d):
			return batch, false
		case line, open := <-streamC:
			if !open {
				return batch, true
			}

			batch = append(batch, line)
			count += 1

			if count == size {
				return batch, false
			}
		}
	}
}

// returns AuthError if user with username is not authorized to take action
// on a job with job_id
func (s *server) authorize(username string, job_id int64) error {
	// find job
	userJobs, exists := s.jobs[username]
	if !exists {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	_, exists = userJobs[job_id]
	if !exists {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	return nil
}
