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

func NewJob(command string) *Job {
	INFO.Printf("Creating a new Job with command: %v\n", command)
	return &Job{}
}

func (j *Job) Start() {
	INFO.Printf("Starting the Job: %v\n", j.Cmd.String())
}

func (j *Job) Status() JobStatus {
	INFO.Printf("Requested for Job Status of Job: %v\n", j.Cmd.String())
	return JobStatus{}
}

func (j *Job) Stop() error {
	INFO.Printf("Requested to Stop Job: %v\n", j.Cmd.String())
	return nil
}

func (j *Job) Output() chan string {
	INFO.Printf("Requested for Output of Job: %v\n", j.Cmd.String())
	return nil
}
