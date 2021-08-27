package job

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var INFO *log.Logger

func init() {
	INFO = log.New(os.Stdout, "LIB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type Job struct {
	Cmd *exec.Cmd
	Pid int

	*sync.Mutex // to protect some of data from simultaneous reads and writes
	outStreamer Streamer
	status      JobStatus // current status of this job
	doneC       chan struct{}
}

type JobStatus struct {
	Status   string // string representation of the status
	Exited   bool   // job as exited (true) or not (false)
	ExitCode int    // exit code of the job; meaningless if not exited
}

// String returns text describing current state of the job
func (js JobStatus) String() string {
	return js.Status
}

type Streamer interface {
	io.WriteCloser
	NewReader() (io.ReadCloser, error)
}

type streamer struct {
	*os.File
}

// NewReader returns a ReadCloser
func (s *streamer) NewReader() (io.ReadCloser, error) {
	f, err := os.Open(s.Name())
	if err != nil {
		return nil, err
	}
	return f, nil
}

// NewStreamer creates a streamer by creating a new file in /tmp
func NewStreamer() (Streamer, error) {
	f, err := ioutil.TempFile("/tmp/", "job.*.log")
	if err != nil {
		return nil, err
	}
	log.Printf("Created tmp file: %v\n", f.Name())
	return &streamer{File: f}, nil
}

// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job {

	args := []string{"/bin/sh", "-c", command}
	job := &Job{
		Cmd:   exec.Command(args[0], args[1:]...),
		Mutex: &sync.Mutex{},
		doneC: make(chan struct{}),
		status: JobStatus{
			Status:   "not running",
			Exited:   false,
			ExitCode: 0,
		},
	}

	return job
}

// Start starts the job. It will call Start() and Wait() on underlying
// exec.Cmd. Any error will be reported to job's status
func (j *Job) Start() {

	// Don't do anything if job has already started
	if j.Cmd.Process != nil {
		return
	}

	// If creating a file to store output fails, don't run the job
	// TODO: this is not ideal solution. Ideally, we want to fall
	// back to in-memory buffer for holding the output.
	// and stream by reading the buffer.
	// TODO: implement the this if have some time
	outStreamer, err := NewStreamer()
	if err != nil {
		INFO.Printf("Failed to create a file to stream output of the run")
		return
	}

	j.outStreamer = outStreamer

	go j.start()
}

// Status returns status of this job.
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {
	return j.status
}

// Stop terminates a running job. Does nothing if job has already stopped.
func (j *Job) Stop() error {
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
	rd, err := j.outStreamer.NewReader()
	if err != nil {
		INFO.Printf("Could not create a reader for the output file")
		return nil
	}

	// create streaming channel and start pushing
	stream := make(chan string)
	go j.startStream(rd, stream)
	return stream
}

func (j *Job) startStream(rd io.ReadCloser, stream chan string) {
	defer close(stream)
	brd := bufio.NewReader(rd)
	for {
		line, err := brd.ReadString('\n')
		if err == io.EOF {
			if j.Status().Exited {
				break
			}
			// at this point, job is still running even though we
			// reach end of the file. that means process will
			// write more information, so we have to wait.
			// instead of reading again immediately, it is better
			// to sleep for an unnoticeable time to prevent CPU heating up.
			time.Sleep(time.Millisecond)
			continue
		}
		stream <- line
	}
}

// Wait blocks until Job is complete.
func (j *Job) Wait() {
	<-j.doneC
}

func (j *Job) start() {
	var err error
	defer func() {
		j.Lock()
		j.status.Exited = true

		if j.Cmd.ProcessState != nil {
			j.status.Status = j.Cmd.ProcessState.String()
			j.status.ExitCode = j.Cmd.ProcessState.ExitCode()
		} else if err != nil {
			j.status.Status = err.Error()
			j.status.ExitCode = -1
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
	j.Cmd.Stdout = j.outStreamer
	j.Cmd.Stderr = j.outStreamer

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
}
