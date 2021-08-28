package main

import (
	"context"
	"log"
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
		return err
	}

	job_id := resp.GetJobId()

	log.Printf("Successfully started Job. Job ID %v\n", job_id)

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

	log.Println(resp.Message)
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
