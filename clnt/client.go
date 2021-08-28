package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	pb "simple-worker/protobuf"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

//  Start command
type StartCommand struct {
	Command string
}

func (cc *StartCommand) run(c *kingpin.ParseContext) error {

	conn, client, err := connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Start(ctx, &pb.StartRequest{
		Command: cc.Command,
	})

	if err != nil {
		if resp != nil {
			err = errors.New(fmt.Sprintf("%v", resp.GetJobId()) + err.Error())
		}
		return err
	}

	job_id := resp.GetJobId()

	fmt.Printf("Successfully started Job. Job ID %v\n", job_id)

	return nil
}

func configureStartCommand(app *kingpin.Application) {
	c := &StartCommand{}
	cc := app.Command("start", "Start a job to execute a command").Action(c.run)
	cc.Flag("command", "Execute specified command in remote server").Required().Short('c').StringVar(&c.Command)
}

//  Stop command
type StopCommand struct {
	JobId int
}

func (cc *StopCommand) run(c *kingpin.ParseContext) error {
	conn, client, err := connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Stop(ctx, &pb.StopRequest{
		JobId: int64(cc.JobId),
	})

	if err != nil {
		return err
	}

	fmt.Println(resp.Message)
	return nil
}

func configureStopCommand(app *kingpin.Application) {
	c := &StopCommand{}
	cc := app.Command("stop", "Stop a job at remote server").Action(c.run)
	cc.Flag("job", "Job ID of the job").Required().Short('j').IntVar(&c.JobId)
}

//  Status command
type StatusCommand struct {
	JobId int
}

func (cc *StatusCommand) run(c *kingpin.ParseContext) error {
	conn, client, err := connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Status(ctx, &pb.StatusRequest{
		JobId: int64(cc.JobId),
	})

	if err != nil {
		return err
	}

	job_id := resp.GetJobId()
	status := resp.GetStatus()
	exited := resp.GetExited()
	exitCode := resp.GetExitCode()

	if !exited {
		fmt.Printf("Job %v: %v\n", job_id, status)
	} else {
		fmt.Printf("Job %v: %v, Exit Code %v\n", job_id, status, exitCode)
	}
	return nil
}

func configureStatusCommand(app *kingpin.Application) {
	c := &StatusCommand{}
	cc := app.Command("status", "Get status of a job at remote server").Action(c.run)
	cc.Flag("job", "Job ID of the job").Required().Short('j').IntVar(&c.JobId)
}

//  Output command
type OutputCommand struct {
	JobId int
}

func (cc *OutputCommand) run(c *kingpin.ParseContext) error {
	conn, client, err := connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Output(ctx, &pb.OutputRequest{
		JobId: int64(cc.JobId),
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
			fmt.Print(line)
		}
	}
	return nil
}

func configureOutputCommand(app *kingpin.Application) {
	c := &OutputCommand{}
	cc := app.Command("output", "Get output of a job at remote server").Action(c.run)
	cc.Flag("job_id", "Job ID of the job").Required().Short('j').IntVar(&c.JobId)
}

// Server connection details
const (
	serverAddr = "127.0.0.1"
	serverPort = "50051"
)

func connect() (*grpc.ClientConn, pb.WorkerClient, error) {
	server := net.JoinHostPort(serverAddr, serverPort)

	// setup insecure connection for now
	// TODO: change this to use mTLS instead
	conn, err := grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewWorkerClient(conn)

	return conn, client, nil
}

func main() {
	app := kingpin.New("client", "Client application to schedule jobs in remote server")
	configureStartCommand(app)
	configureStopCommand(app)
	configureStatusCommand(app)
	configureOutputCommand(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
