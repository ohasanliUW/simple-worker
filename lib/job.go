package job

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

var INFO *log.Logger

func init() {
	INFO = log.New(os.Stdout, "LIB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type Job struct {
	Cmd *exec.Cmd
	Pid int

	*sync.Mutex           // to protect some of data from simultaneous reads and writes
	status      JobStatus // current status of this job
	doneC       chan struct{}

	// TODO: Add output Streamer
}

type JobStatus struct {
	Status   string // string representation of the status
	Exited   bool   // job as exited (true) or not (false)
	ExitCode int    // exit code of the job; meaningless if not exited
}

func (js JobStatus) String() string {
	return js.Status
}

// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job {
	INFO.Printf("Creating a new Job with command: %v\n", command)

	args := []string{"/bin/sh", "-c", command}

	job := &Job{
		Cmd:   exec.Command(args[0], args[1:]...),
		Mutex: &sync.Mutex{},
		doneC: make(chan struct{}),
	}

	return job
}

// Start starts the job. It will call Start() and Wait() on underlying
// exec.Cmd. Any error will be reported to job's status
func (j *Job) Start() {
	INFO.Printf("Starting the Job: %v\n", j.Cmd.String())

	if j.Cmd.Process != nil {
		// This job has already started, do nothing
		return
	}

	go func() {
		var err error
		defer func() {
			j.Lock()
			j.status.Exited = true

			if j.Cmd.ProcessState != nil {
				j.status.Status = j.Cmd.ProcessState.String()
				j.status.ExitCode = j.Cmd.ProcessState.ExitCode()
			} else {
				j.status.Status = "Unknown Error, could not start the process"
				j.status.ExitCode = -1
			}

			j.Unlock()
			// close the channel to indicate that Job has finished
			close(j.doneC)
		}()

		// following creates a process group, so that killing parent would kill its child processes
		j.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// TODO: change Cmd.Stdout and Cmd.Stderr to write all of the output into a file

		err = j.Cmd.Start()
		if err != nil {
			INFO.Println("Start() failed:", err)
			return
		}

		j.Lock()
		j.status.Status = "running"
		j.status.Exited = false
		j.Pid = j.Cmd.Process.Pid
		j.Unlock()

		INFO.Println("Successfully started job:", j.Pid)

		err = j.Cmd.Wait()
		if err != nil {
			INFO.Println("Wait() returned with an error:", err)
			return
		}
	}()
}

// Status returns status of this job.
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {
	INFO.Printf("Requested for Job Status of Job: %v\n", j.Cmd.String())
	return j.status
}

// Stop terminates a running job. Does nothing if job has already stopped.
func (j *Job) Stop() error {
	INFO.Printf("Requested to Stop Job: %v\n", j.Cmd.String())
	pgid, err := syscall.Getpgid(j.Pid)
	if err != nil {
		INFO.Printf("Unable to get group id for process %v\n", j.Pid)
		return err
	}

	err = syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		INFO.Printf("Unable to kill process group")
		return err
	}

	return err
}

// Output returns a string channel which will be closed by library internally
// once the job completes. Multiple calls to this function will create separate channels
// so it is safe to stream output to multiple clients.
func (j *Job) Output() chan string {
	INFO.Printf("Requested for Output of Job: %v\n", j.Cmd.String())
	return nil
}

// Wait blocks until Job is complete.
func (j *Job) Wait() {
	<-j.doneC
}
