package job

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Job struct {
	*sync.Mutex // to protect some of data from simultaneous reads and writes
	id          int64
	cmd         *exec.Cmd
	outStreamer Streamer
	status      JobStatus // current status of this job
	doneC       chan struct{}
}

type JobStatus struct {
	Status   string // string representation of the status
	Exited   bool   // job as exited (true) or not (false)
	ExitCode int    // exit code of the job; meaningless if not exited

	started bool // flag to indicate that Start() was successful
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

// NewReader returns a ReadCloser for the output file this streamer is using
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
	return &streamer{File: f}, nil
}

// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job {
	args := strings.Split(command, " ")
	job := &Job{
		cmd:   exec.Command(args[0], args[1:]...),
		id:    time.Now().Unix(),
		Mutex: &sync.Mutex{},
		doneC: make(chan struct{}),
		status: JobStatus{
			Status:   "not running",
			Exited:   false,
			ExitCode: 0,
			started:  false,
		},
	}

	return job
}

// Start starts the job. It will call Start() and Wait() on underlying
// exec.Cmd. Any error will be reported to job's status
func (j *Job) Start() error {

	// If job already started, return an error indicating so
	if j.cmd.Process != nil {
		return errors.New("job has already started")
	}

	// If job has already exited, then disallow rerun
	// NOTE: user can always create a new job with same command
	if j.status.Exited {
		return errors.New("job has exited; cannot restart")
	}

	// If creating a file to store output fails, don't run the job
	// TODO: this is not ideal solution. Ideally, we want to fall
	// back to in-memory buffer for holding the output.
	// and stream by reading the buffer.
	// TODO: implement the this if have some time
	outStreamer, err := NewStreamer()
	if err != nil {
		return errors.New("failed to create a file to stream output of the run")
	}

	j.outStreamer = outStreamer

	// start() will call exec.Cmd.Start() which might return an error
	// need to wait until result of Start() comes back so that we can
	// immediately report it back to the caller
	errC := make(chan error)
	go j.start(errC)

	// Channel will either return an error or will be closed and give nil error
	// In any case, we need to return it back to the caller
	err = <-errC

	return err
}

// Status returns status of this job.
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {
	return j.status
}

// Stop terminates a running job. Returns immediately if job has already exited
func (j *Job) Stop() error {

	if j.status.Exited {
		return nil
	}

	// if job has not started yet, simply mark it exited
	if !j.status.started {
		j.status.Exited = true

	}

	pid, err := j.Pid()
	if err != nil {
		return err
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}

	err = syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
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
		return nil
	}

	// create streaming channel and start pushing
	stream := make(chan string)
	go j.startStream(rd, stream)
	return stream
}

// Wait blocks until Job is complete.
func (j *Job) Wait() {
	<-j.doneC
}

// Id returns the job id
func (j *Job) Id() int64 {
	return j.id
}

// Pid returns the PID of underlying process
func (j *Job) Pid() (int, error) {
	if j.cmd.Process == nil {
		return -1, errors.New("job has not started yet")
	}
	return j.cmd.Process.Pid, nil
}

// startStream takes a ReadCloser and reads data line by line
// then pushes them into the provided channel until Job is exited
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

// start calls Start() and Wait() of underlying exec.Cmd
// it also updates job status accordingly
// once Wait() returns, it closes doneC channel so that
// users can synchronize using this channel
func (j *Job) start(errC chan error) {
	var err error
	defer func() {
		j.Lock()
		j.status.Exited = true

		if j.cmd.ProcessState != nil {
			j.status.Status = j.cmd.ProcessState.String()
			j.status.ExitCode = j.cmd.ProcessState.ExitCode()
		} else if err != nil {
			j.status.Status = err.Error()
			j.status.ExitCode = -1
		} else {
			j.status.Status = "completed"
			j.status.ExitCode = 0
		}

		j.Unlock()
		// close the channel to indicate that Job has finished
		close(j.doneC)
	}()

	// following creates a process group, so that killing parent would kill its child processes
	j.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set both stdout and stderr of the command to use our new streamer so that combined output
	// goes into a single place
	j.cmd.Stdout = j.outStreamer
	j.cmd.Stderr = j.outStreamer

	err = j.cmd.Start()
	if err != nil {
		errC <- err
		close(errC)
		return
	}

	// Start did not fail, so just close the errC
	close(errC)

	j.Lock()
	j.status.Status = "running"
	j.status.Exited = false
	j.status.started = true
	j.Unlock()

	// Note: errC error channel will not be used for Wait(). Purpose of
	// errC is to return an immediate error from Start().
	err = j.cmd.Wait()
	if err != nil {
		return
	}
}
