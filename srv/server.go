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
	"strings"
	"syscall"
	"time"

	pb "simple-worker/protobuf"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedWorkerServer
	jobs map[uuid.UUID]*lib.Job // all jobs created by server
}

// a custom error that will be returned when a request is not authorized
type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}

const (
	port = ":50051"
)

func main() {
	srv := &server{
		jobs: make(map[uuid.UUID]*lib.Job),
	}

	gsrv := grpc.NewServer(
		grpc.Creds(loadKeyPair()),
		grpc.UnaryInterceptor(middlefunc),
	)

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

// Start receives an RPC containing a command to execute.
// On success, it will start the execution and respond with job id
// On failure, it will return an error
func (s *server) Start(ctx context.Context, in *pb.StartRequest) (*pb.StartResponse, error) {
	command := in.GetCommand()

	if command == "" {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"must provide a valid command",
		)
	}

	username, err := getUsername(ctx)
	if err != nil {
		return nil, err
	}

	job, err := s.startJob(username, command)
	if err != nil {
		// if job fails during exec.Cmd.Start(), then there is a
		// job id associated with it. Add this id to error message
		// before returning it
		if job != nil {
			err = errors.New(fmt.Sprintf("Job with ID %v: %v", job.UUID.String(), err.Error()))
		}
		return nil, err
	}

	return &pb.StartResponse{
		JobId: job.UUID[:],
	}, nil
}

// Stop receives an RPC containing job id of the job to stop.
// On success, job is stopped and client is responded with a message describing it
// On failure, it will return an error
func (s *server) Stop(ctx context.Context, in *pb.StopRequest) (*pb.StopResponse, error) {

	job_id := in.GetJobId()
	username, err := getUsername(ctx)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	if err = id.UnmarshalBinary(job_id); err != nil {
		return nil, err
	}

	err = s.stopJob(username, id)
	if err != nil {
		return nil, err
	}

	return &pb.StopResponse{
		Message: fmt.Sprintf("Successfully stopped job %v", id.String()),
	}, nil
}

// Status receives an RPC containing job id of the job to query
// On success, it will respond to client with UUID, Status and ExitCode of the job
// On failure, it will return an error
func (s *server) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusResponse, error) {
	job_id := in.GetJobId()
	username, err := getUsername(ctx)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	if err = id.UnmarshalBinary(job_id); err != nil {
		return nil, err
	}

	status, err := s.statusOfJob(username, id)
	if err != nil {
		return nil, err
	}

	return &pb.StatusResponse{
		JobId:    job_id,
		Status:   status.String(),
		ExitCode: int32(status.ExitCode),
	}, nil
}

// Output receives an RPC containing  job id of the job
// On success, it will start streaming output of the job to the client
// On failure, it will return an error
func (s *server) Output(in *pb.OutputRequest, stream pb.Worker_OutputServer) (err error) {
	job_id := in.GetJobId()
	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}

	id := uuid.New()
	if err = id.UnmarshalBinary(job_id); err != nil {
		return err
	}

	streamC, err := s.outputOfJob(username, id)
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

	// Special check whether UUID exists already. Disallow if so as we
	// should never hit this (almost never: see uuid.NewRandom())
	// TODO: Should handle this possibility by checking the existing list
	// of jobs. As jobs are not managed by the library, I will skip this
	// handling for now. Might need to handle the jobs internally in library
	// to resolve this in a cleaner fashion.
	_, exists := s.jobs[job.UUID]
	if exists {
		return nil, errors.New("server had a hick-up. Can you try again?")
	}

	// set the username for this job. All other RPCs will check this field
	// for authorizing the requested service for this job
	job.SetUsername(username)
	s.jobs[job.UUID] = job
	if err := job.Start(); err != nil {
		return job, err
	}

	return job, nil
}

// stopJob attempts to stop the job with job_id. if username does not match
// the username in this job, it will return an authorization error.
func (s *server) stopJob(username string, job_id uuid.UUID) error {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return authErr
	}

	job := s.jobs[job_id]
	err := job.Stop()
	return err
}

// statusOfJob will return status of specified job. If username does not match
// the username in this job, it will return an authorization error
func (s *server) statusOfJob(username string, job_id uuid.UUID) (lib.JobStatus, error) {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return lib.JobStatus{}, authErr
	}

	job := s.jobs[job_id]
	status := job.Status()
	return status, nil
}

// outputOfJob will return a channel of byte slices containing output of the job.
// If username does not match the username in this job, it will return an authorization error
func (s *server) outputOfJob(username string, job_id uuid.UUID) (chan []byte, error) {
	authErr := s.authorize(username, job_id)
	if authErr != nil {
		return nil, authErr
	}

	job := s.jobs[job_id]
	outStreamC := job.Output()

	if outStreamC == nil {
		return nil, errors.New("could not get output of the job")
	}

	return outStreamC, nil
}

// getNextLineBatch takes a channel of byte slices, number of max lines per batch needed,
// and a delay to back off from waiting on channel, and returns slice of byte slices and
// a boolean representing whether stream channel is closed
func getNextLineBatch(streamC chan []byte, size int, d time.Duration) ([][]byte, bool) {
	batch := make([][]byte, 0, size)
	count := 0

	for {
		select {
		case <-time.After(d):
			// couldn't get data in time, return whatever we have
			// but indicate that we expect some more
			return batch, false
		case line, open := <-streamC:
			if !open {
				// received a signal that channel is closed. this means
				// job has terminated  and no more output should be expected
				// return whatever we have and indicate the end
				return batch, true
			}

			// received some data. add it to the batch
			batch = append(batch, line)
			count += 1

			// if batch is full, return it but indicate that there are some more
			if count == size {
				return batch, false
			}
		}
	}
}

// get client name embedded into Common Name field for authorization
func getUsername(ctx context.Context) (string, error) {
	peer, ok := peer.FromContext(ctx)
	if ok {
		tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
		cn := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
		return cn, nil
	}
	return "", errors.New("common name is missing in client certificate")
}

// returns AuthError if user with username is not authorized to take action
// on a job with job_id
func (s *server) authorize(username string, job_id uuid.UUID) error {
	// find job
	job, exists := s.jobs[job_id]
	if !exists {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	if strings.Compare(job.Username(), username) != 0 {
		return &AuthError{
			fmt.Sprintf("job id %v not found", job_id),
		}
	}

	return nil
}

func middlefunc(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// get client tls info
	if p, ok := peer.FromContext(ctx); ok {
		if mtls, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			for _, item := range mtls.State.PeerCertificates {
				log.Println("request certificate subject:", item.Subject)
			}
		}
	}
	return handler(ctx, req)
}
