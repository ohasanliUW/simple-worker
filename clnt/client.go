package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

//  Start command
type StartCommand struct {
	Command string
}

func (cc *StartCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("command=%v\n", cc.Command)
	return nil
}

func configureStartCommand(app *kingpin.Application) {
	c := &StartCommand{}
	cc := app.Command("start", "Start a job to execute a command(s)").Action(c.run)
	cc.Flag("command", "Execute specified command in remote server (Repeatable)").Required().Short('c').StringVar(&c.Command)
}

//  Stop command
type StopCommand struct {
	PID int
}

func (cc *StopCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("PID=%v\n", cc.PID)
	return nil
}

func configureStopCommand(app *kingpin.Application) {
	c := &StopCommand{}
	cc := app.Command("stop", "Stop a job(s) at remote server").Action(c.run)
	cc.Flag("pid", "Process ID of the job (Repeatable)").Required().Short('p').IntVar(&c.PID)
}

//  Status command
type StatusCommand struct {
	PID []int
}

func (cc *StatusCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("PID=%v\n", cc.PID)
	return nil
}

func configureStatusCommand(app *kingpin.Application) {
	c := &StatusCommand{}
	cc := app.Command("status", "Get status of a job(s) at remote server").Action(c.run)
	cc.Flag("pid", "Process ID of the job (Repeatable)").Required().Short('p').IntsVar(&c.PIDs)
}

//  Output command
type OutputCommand struct {
	PID int
}

func (cc *OutputCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("PID=%v\n", cc.PID)
	return nil
}

func configureOutputCommand(app *kingpin.Application) {
	c := &OutputCommand{}
	cc := app.Command("output", "Get output of a job at remote server").Action(c.run)
	cc.Flag("pid", "Process ID of the job").Required().Short('p').IntVar(&c.PID)
}

// Server connection details
const (
	serverAddr = "127.0.0.1"
	serverPort = "50051"
)

func connect() error {
	// server := net.JoinHostPort(serverAddr, serverPort)

	// setup insecure connection for now
	// TODO: change this to use mTLS instead
	//conn, err := grpc.Dial(server, grpc.WithInsecure())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	return nil
}

func main() {
	app := kingpin.New("client", "Client application to schedule jobs in remote server")
	configureStartCommand(app)
	configureStopCommand(app)
	configureStatusCommand(app)
	configureOutputCommand(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
