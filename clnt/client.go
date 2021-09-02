package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	pb "simple-worker/protobuf"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/alecthomas/kingpin.v2"
)

//  Start command
type StartCommand struct {
	Command string
	client  pb.WorkerClient
}

// takes parsed command line arguments and makes a Start() RPC call to the server
func (cc *StartCommand) run(c *kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cc.client.Start(ctx, &pb.StartRequest{
		Command: cc.Command,
	})

	if err != nil {
		if resp != nil {
			err = fmt.Errorf("%v: %v", resp.GetJobId(), err.Error())
		}
		return err
	}

	job_id := resp.GetJobId()

	id := uuid.New()
	if err = id.UnmarshalBinary(job_id); err != nil {
		return err
	}
	fmt.Printf("Successfully started Job. Job ID %v\n", id.String())

	return nil
}

func configureStartCommand(app *kingpin.Application, client pb.WorkerClient) {
	c := &StartCommand{
		client: client,
	}
	cc := app.Command("start", "Start a job to execute a command").Action(c.run)
	cc.Flag("command", "Execute specified command in remote server").Required().Short('c').StringVar(&c.Command)
}

//  Stop command
type StopCommand struct {
	JobId  string
	client pb.WorkerClient
}

// takes parsed command line arguments and makes a Stop() RPC call to the server
func (cc *StopCommand) run(c *kingpin.ParseContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := uuid.New()
	if err := id.UnmarshalText([]byte(cc.JobId)); err != nil {
		return err
	}

	resp, err := cc.client.Stop(ctx, &pb.StopRequest{
		JobId: id[:],
	})

	if err != nil {
		return err
	}

	fmt.Println(resp.Message)
	return nil
}

func configureStopCommand(app *kingpin.Application, client pb.WorkerClient) {
	c := &StopCommand{
		client: client,
	}
	cc := app.Command("stop", "Stop a job at remote server").Action(c.run)
	cc.Flag("job", "Job ID of the job").Required().Short('j').StringVar(&c.JobId)
}

//  Status command
type StatusCommand struct {
	JobId  string
	client pb.WorkerClient
}

// takes parsed command line arguments and makes a Status() RPC call to the server
func (cc *StatusCommand) run(c *kingpin.ParseContext) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := uuid.New()
	if err := id.UnmarshalText([]byte(cc.JobId)); err != nil {
		return err
	}

	resp, err := cc.client.Status(ctx, &pb.StatusRequest{
		JobId: id[:],
	})

	if err != nil {
		return err
	}

	job_id := resp.GetJobId()
	status := resp.GetStatus()
	exitCode := resp.GetExitCode()

	id = uuid.New()
	if err = id.UnmarshalBinary(job_id); err != nil {
		return err
	}

	if status == "running" {
		fmt.Printf("Job %v: %v\n", id.String(), status)
	} else {
		fmt.Printf("Job %v: %v, Exit Code %v\n", id.String(), status, exitCode)
	}
	return nil
}

func configureStatusCommand(app *kingpin.Application, client pb.WorkerClient) {
	c := &StatusCommand{
		client: client,
	}
	cc := app.Command("status", "Get status of a job at remote server").Action(c.run)
	cc.Flag("job", "Job ID of the job").Required().Short('j').StringVar(&c.JobId)
}

//  Output command
type OutputCommand struct {
	JobId  string
	client pb.WorkerClient
}

// takes parsed command line arguments and makes an Output() RPC call to the server
func (cc *OutputCommand) run(c *kingpin.ParseContext) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	id := uuid.New()
	if err := id.UnmarshalText([]byte(cc.JobId)); err != nil {
		return err
	}

	stream, err := cc.client.Output(ctx, &pb.OutputRequest{
		JobId: id[:],
	})

	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		// if EOF, simply break the loop and return nil
		if err == io.EOF {
			break
		}

		// if err, return it
		if err != nil {
			return err
		}

		// for each received line, simply print it
		for _, line := range resp.OutputLine {
			fmt.Print(string(line))
		}
	}
	return nil
}

func configureOutputCommand(app *kingpin.Application, client pb.WorkerClient) {
	c := &OutputCommand{
		client: client,
	}
	cc := app.Command("output", "Get output of a job at remote server").Action(c.run)
	cc.Flag("job", "Job ID of the job").Required().Short('j').StringVar(&c.JobId)
}

// Server connection details
const (
	serverAddr = "localhost"
	serverPort = "50051"
)

func connect() (*grpc.ClientConn, pb.WorkerClient, error) {
	server := net.JoinHostPort(serverAddr, serverPort)

	conn, err := grpc.Dial(server, grpc.WithTransportCredentials(loadKeyPair()))
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewWorkerClient(conn)

	return conn, client, nil
}

// function to load certificate and private key pair
// TODO: this function uses hard coded values for key pairs.
func loadKeyPair() credentials.TransportCredentials {
	certPath := "certs/client.crt"
	keyPath := "certs/client.key"

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatalln("Failed to load certificate:", certPath, keyPath)
	}

	ca_data, err := ioutil.ReadFile("certs/rootCA.crt")
	if err != nil {
		log.Fatalln("Failed to read root CA data")
	}

	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM(ca_data) {
		log.Fatalln("Failed to add CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      capool,
	}

	return credentials.NewTLS(tlsConfig)
}

func main() {
	conn, client, err := connect()
	if err != nil {
		fmt.Printf("Failed: %v\n", err.Error())
		return
	}
	defer conn.Close()

	app := kingpin.New("client", "Client application to schedule jobs in remote server")
	configureStartCommand(app, client)
	configureStopCommand(app, client)
	configureStatusCommand(app, client)
	configureOutputCommand(app, client)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
