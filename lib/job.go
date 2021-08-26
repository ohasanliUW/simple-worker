package job

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

var INFO *log.Logger

func init() {
	INFO = log.New(os.Stdout, "LIB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type Job struct {
	Cmd *exec.Cmd
	Pid int

	*sync.Mutex // to protect some of data from simultaneous reads and writes
	//status      JobStatus // current status of this job
	//doneC       chan struct{}

	// TODO: Add output Streamer
}

type JobStatus struct {
	Status   string // string representation of the status
	Exited   bool   // job as exited (true) or not (false)
	ExitCode int    // exit code of the job; meaningless if not exited
}

// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job {
	INFO.Printf("Creating a new Job with command: %v\n", command)
	return &Job{}
}

// Start starts the job. It will call Start() and Wait() on underlying
// exec.Cmd. Any error will be reported to job's status
func (j *Job) Start() {
	INFO.Printf("Starting the Job: %v\n", j.Cmd.String())
}

// Status returns status of this job.
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {
	INFO.Printf("Requested for Job Status of Job: %v\n", j.Cmd.String())
	return JobStatus{}
}

// Stop terminates a running job. Does nothing if job has already stopped.
func (j *Job) Stop() error {
	INFO.Printf("Requested to Stop Job: %v\n", j.Cmd.String())
	return nil
}

// Output returns a string channel which will be closed by library internally
// once the job completes. Multiple calls to this function will create separate channels
// so it is safe to stream output to multiple clients.
func (j *Job) Output() chan string {
	INFO.Printf("Requested for Output of Job: %v\n", j.Cmd.String())
	return nil
}
