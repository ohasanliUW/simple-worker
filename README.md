

# Code Challenge - Teleport

Purpose of this project is to create a small service that is able to `Start`, `Stop`, get `Status` and stream `Output` of a Linux job. The complete package will consist of 3 main components:  
1. Library to achieve above functionalities (used by server)  
1. gRPC to achieve communication between clients and the server 
1. CLI for clients to send requests to server to use above services


## Library
The library will provide functions to create, stop, query status and stream output of a job. It will use `os/exec` package to achieve most of its functionalities.

The library will **not** store created jobs internally in memory. It will simply return the `Job` to the user and it is up to the user to manage them. This library will provide all the utilities needed to `Create`, `Stop`, `GetStatus`, and `Output` of a job.

### Data structures
```go
type Job struct {
	Cmd    *exec.Cmd
	Pid    int

	status JobStatus	// Status of this Job
	*sync.Mutex			// to protect Status from simultaneous reads and writes
	outStream Streamer	// used for creating streaming channel via Output()
	doneC     chan struct{} // channel for user to block until job exits
}

type JobStatus {
	Status string  // string representation of current status
	Exited bool    // true if job has exited
	ExitCode int   // exit code of the exited job
	// extend as needed
}

type Streamer interface {
	io.WriteCloser	// implements Writer and Closer
	NewReader() (io.ReadCloser, error)
}

// streamer structure embeds os.File for implementing
// simultaneous output streaming for multiple clients.
type streamer struct {
	*os.File // embeds os.File
}

```
  Note that `streamer` structure embeds `os.File` and has many benefits:
  1. Read() and Write() methods for free
  2. Synchronization by kernel for free
  3. Memory error safe as data is stored in disk
  4. Will be signed to exec.Cmd.Stdout and exec.Cmd.Stderr fields and both outputs will go into a single file in chronological order.

Say, if a client is requesting for a job that builds a project, logs can be very long and may take up a lof of space. The above approach is very handy as it uses disk space rather than in-memory buffers. And implementation will be simple as many things come for free.

  
  To keep it simple, this library will do followings which **might not** be suitable for production:
1. Execute all provided commands only through `/bin/sh`.
2. Server (as library user) will track all jobs created in-memory via a map data structure. To make it more practicle, other options can be used (database, file system, infra provided by kernel, etc.). In my opinion, for *Unix* and *Linux*, `/proc/` directory has been one of the most consistent performers for job/task/process monitoring.

### Using library
Following are the functions this library will provide
```go
// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job
// Start starts the job. It will call Start() and Wait() on underlying
// exec.Cmd. Any error will be reported to job's status. (see data structures)
func (j* Job) Start() {}
// Status returns status of this job. See struct JobStatus for more details
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {}
// Stop terminates a running job. Does nothing if job has already stopped.
func (j *Job) Stop() error {}
// Output returns a string channel which will be closed by library internally
// once the job completes. Multiple calls to this function will create separate channels
// so it is safe to stream output to multiple clients.
func (j *Job) Output() chan string {}
```

## gRPC
To implement connectivity between clients and the server, gRPC package will be used.
Following proto file will be used for RPCs.
```go
service Worker {
  // Starts a job
  rpc Start(stream StartRequest) returns (stream StatusResponse) {}
  // Stops the specified job
  rpc Stop(stream StopRequest) returns (stream StatusResponse) {}
  // Gets status of a job
  rpc Status(stream StatusRequest) returns (stream StatusResponse) {}
  // List all jobs
  rpc Output(OutputRequest) returns (stream OutputResponse) {}
}

message StartRequest {
  string command = 1;
}

message StopRequest {
  int32 job_id = 1;
}

message StatusRequest {
  int32 job_id = 1;
}

message StatusResponse {
  int32 job_id = 1;
  string status = 2;
  bool exited = 3;
  int32 exit_code = 4;
}

message OutputRequest {
  int32 job_id = 1;
}

message OutputResponse {
  string output_line = 1;
}
```
`Start` will stream various commands to server and server will create `Job` for each command and stream back a status for each command.
`Stop`, similarly to `Start`, will stream list of pids/job ids to server and server will respond back with a stream of job statuses after it kills every single requested job.
`Status`, as well, will stream list of pids/job ids to server and server will respond back with a stream of job statuses.
`Output` will request for output of a single job and server will stream its output back line by line.
Note: It will always stream from the start of the job; no matter how many times requested. If job is complete, it will stream all the way to the end. If job is still running, it will wait for new data to arrive until server completes the stream. Behaviour will be similar to of `tail -f`.


### Authentication
Communication channels between clients and server will be protected via mutual TLS (mTLS). For simplicity, I will create a root CA and sign certificate for both server and its clients. Clients and the server will use these generated static certificates. Server and clients will authenticate each other by requesting other party's certificate and checking it with CA.

### Authorization Scheme
For simplicity, each client certificate will embed client's username in `Common Name` field. Server. will extract this information and authorize RPCs on existing jobs if and only if it belongs to that user.
Server will record each created Job in a map data structure. It will look like this:
`map[string]map[int]*Job`
I.e, username -> JobId/PID -> Job

## Client CLI
Client will be a simple go program that will connect to server by using gRPC as described in above gRPC section. It will be implemented via kingpin.v2 package.
Command will look like this:
```
usage: client [<flags>] <command> [<args> ...]

Client application to schedule jobs in remote server

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.

  start --command=COMMAND
    Start a job to execute a command(s)

  stop --pid=PID
    Stop a job(s) at remote server

  status --pid=PID
    Get status of a job(s) at remote server

  output --pid=PID
    Get output of a job at remote server

```

### Usage
1. Start:	 `client start -c "echo Hello World" -c "echo Hello Again"`
2. Status: `client status -p 1234 -p 1235 -p 1236`
3. Stop:    `client stop -p 1234 -p 1235 -p 1236`
4. Output:`client output -p 1234`

## Testing
All test cases will be implemented via packages `gopkg.in/check.v1` and `testing`.
### Library:
For each of the exported functions, there will be a test to cover positive case. For some of them, I will add a negative test case. This is only for demonstrating tests could be implemented to get higher code coverage.
### gRPC and server
One test case for successful authentication
One test case for failed authentication
One test case for successful authorization
One test case for failed authorization
One test case for networking (RPC can be sent and received)

