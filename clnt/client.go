package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

///////////////////////////////
//  START COMMAND
///////////////////////////////
type StartCommand struct {
	Command []string
}

func (cc *StartCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("command=%v\n", cc.Command)
	for _, cc := range cc.Command {
		fmt.Printf("Command: %v\n", cc)
	}
	return nil
}

func configureStartCommand(app *kingpin.Application) {
	c := &StartCommand{}
	cc := app.Command("start", "Start a job to execute a command(s)").Action(c.run)
	cc.Flag("command", "Execute specified command in remote server (Repeatable)").Required().Short('c').StringsVar(&c.Command)
}

///////////////////////////////
//  STOP COMMAND
///////////////////////////////
type StopCommand struct {
	PIDs []int
}

func (cc *StopCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("PIDs=%v\n", cc.PIDs)
	for _, cc := range cc.PIDs {
		fmt.Printf("PID: %v\n", cc)
	}
	return nil
}

func configureStopCommand(app *kingpin.Application) {
	c := &StopCommand{}
	cc := app.Command("stop", "Stop a job(s) at remote server").Action(c.run)
	cc.Flag("pid", "Process ID of the job (Repeatable)").Required().Short('p').IntsVar(&c.PIDs)
}

///////////////////////////////
//  STATUS COMMAND
///////////////////////////////
type StatusCommand struct {
	PIDs []int
}

func (cc *StatusCommand) run(c *kingpin.ParseContext) error {
	fmt.Printf("PIDs=%v\n", cc.PIDs)
	for _, cc := range cc.PIDs {
		fmt.Printf("PID: %v\n", cc)
	}
	return nil
}

func configureStatusCommand(app *kingpin.Application) {
	c := &StatusCommand{}
	cc := app.Command("status", "Get status of a job(s) at remote server").Action(c.run)
	cc.Flag("pid", "Process ID of the job (Repeatable)").Required().Short('p').IntsVar(&c.PIDs)
}

///////////////////////////////
//  OUTPUT COMMAND
///////////////////////////////
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

func main() {
	app := kingpin.New("client", "Client application to schedule jobs in remote server")
	configureStartCommand(app)
	configureStopCommand(app)
	configureStatusCommand(app)
	configureOutputCommand(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
