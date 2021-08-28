
  

# Code Challenge - Teleport

Purpose of this project is to create a small service that is able to `Start`, `Stop`, get `Status` and stream `Output` of a Linux job. The complete package will consist of 3 main components:  
1. Library to achieve above functionalities (used by server)  
1. gRPC to achieve communication between clients and the server 
1. CLI for clients to send requests to server to use above services


## Library
The library will provide functions to create, stop, query status and stream output of a job. It will use `os/exec` package to achieve most of its functionalities.

This library will provide all the utilities needed to `Create`, `Stop`, `GetStatus`, and `Output` of a job. The library will **not** store created jobs internally in memory. It will simply return the `Job` to the user and it is up to the user to manage them. It simplifies the implementation and allows the user to have a flexible job management scheme. However, losing the reference to the job will not allow accessing to that job anymore. I.e, library does not provide any mechanism to list all the existing jobs. However, implementing such mechanism in library in-memory is very simple. This can be achieved by having a global variable of type `map[int]*Job` (protected by a `sync.Mutex`).

### Data structures
```go
type Job struct {
	Id		int64

	cmd    *exec.Cmd
	status JobStatus	// Status of this Job
	*sync.Mutex			// to protect Status from simultaneous reads and writes
	outStream Streamer	// used for creating streaming channel via Output()
	doneC     chan struct{} // channel for user to block until job exits
}

type JobStatus {
	Status string  // string representation of current status
	Exited bool    // true if job has exited
	ExitCode int   // exit code of the exited job

	started bool   // true if Start() was successful
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
  3. Memory-error safe as data is stored in disk
  4. Will be asigned to exec.Cmd.Stdout and exec.Cmd.Stderr fields and both outputs will go into a single file in chronological order.

Say, if a client is requesting for a job that builds a project, logs can be very long and may take up a lof of space. The above approach is very handy as it uses disk space rather than in-memory buffers. And implementation will be simple as many things come for free.
A disadvantage to this approach is that it requires some effort to make the disk space available at all times. A cron job can be used to remove files with certain age from directory where all job outputs are collected.

Server (as library user) will track all jobs created in-memory via a map data structure. To make it more practicle, other options can be used (database, file system, infra provided by kernel, etc.)

### Using library
Following are the functions this library will provide
```go
// NewJob creates a job with specified command but does not start it
func NewJob(command string) *Job
// Start starts the job execution. Calls exec.Cmd.Start() implicitly
// If the call fails, its error is returned by Start()
func (j* Job) Start() error {}
// Status returns status of this job. See struct JobStatus for more details
// Users can determine whether specified job has completed or not via this function
func (j *Job) Status() JobStatus {}
// Stop terminates a running job. Does nothing if job has already stopped.
func (j *Job) Stop() error {}
// Output returns a string channel which will be closed by library internally
// once the job completes. Multiple calls to this function will create separate channels
// so it is safe to stream output to multiple clients.
func (j *Job) Output() chan string {}
// Pid returns PID of process running the job.
// If no such process exist, an error will be returned.
func (j *Job) Pid() (int, error) {}
```

## gRPC
To implement connectivity between clients and the server, gRPC package will be used.
Following proto file will be used for RPCs.
```go
service Worker {
  // Starts a job
  rpc Start(StartRequest) returns (StartResponse) {}
  // Stops the specified job
  rpc Stop(StopRequest) returns (StopResponse) {}
  // Gets status of a job
  rpc Status(StatusRequest) returns (StatusResponse) {}
  // Get output of a job. If job is still running, it will stream
  // the output
  rpc Output(OutputRequest) returns (stream OutputResponse) {}
}

message StartRequest {
  string command = 1;
}

message StartResponse {
  string command = 1;
}

message StopRequest {
  int32 job_id = 1;
}

message StopResponse {
  string message = 1;
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
  repeated string output_line = 1;
}
```
1. **`Start`** will send various commands batched into a single RPC to the server and server will create `Job` for each command and send back a batch of status response.
1. **`Stop`** will and RPC with a single job id to a server and server will stop the specified job if authorization is successful. A response will be sent back with an appropriate message.
`- successfully stopped job with id <id>`
`- error: job id not found`
If authorization fails for a running job, then instead of admitting that "such job exists but failed to authorize for stopping it", it is better to simply to say `job id not found`. Similar to how entering wrong password for a valid email tells that `username or password is incorrect` rather than saying that `password is incorrect`.

1. **`Status`**, will request for currrert status/state of a job and server will send back a response containing its state, exit code and some human readable message (`scheduled`, `running`, `completed`, etc.)
1. **`Output`** will request for output of a single job and server will stream its output back line by line.
Note: It will always stream from the start of the job; no matter how many times requested. If job is complete, it will stream all the way to the end. If job is still running, it will wait for new data to arrive until server completes the stream. Behaviour will be similar to of `tail -f`.
To optimize the performance a little, `OutputResponse` will consist of a string slice `[]string`. Server will try to batch up to `N` lines of output (`N` can be any reasonable number: 10, 20, 30, etc.) and send them with a single RPC response to reduce the number of calls needed over the network to complete the task. If the channel feeding the information does not have `N` lines of output, server will not wait and instead send whatever it has batched into the response.


### Authentication
Communication channels between clients and server will be protected via mutual TLS (mTLS). For simplicity, I will create a root CA and sign certificate for both server and its clients. Clients and the server will use these generated static certificates. Server and clients will authenticate each other by requesting other party's certificate and checking it with CA.

### Authorization Scheme
For simplicity, each client certificate will embed client's username in `Common Name` field. Server. will extract this information and authorize RPCs on existing jobs if and only if it belongs to that user.
Server will record each created Job in a map data structure. It will look like this:
`map[string]map[int]*Job`
I.e, username -> Job ID -> Job

If authorization fails for a running job, then instead of admitting that "such job exists but failed to authorize for stopping it", it is better to simply to say `job id not found`. Similar to how entering wrong password for a valid email tells that `username or password is incorrect` rather than saying that `password is incorrect`.

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
    Start a job to execute a command at remote server

  stop --job=ID
    Stop a job at remote server

  status --job=ID
    Get status of a job at remote server

  output --job=ID
    Get output of a job at remote server

```

### Usage
1. Start:	 `client start -c "echo Hello World"`
2. Status: `client status -j 1234`
3. Stop:    `client stop -j 1234`
4. Output:`client output -j 1234`

## Testing
All test cases will be implemented via packages `gopkg.in/check.v1` and `testing`.
### Library:
For each of the exported functions, there will be a test to cover positive case. For some of them, I will add a negative test case to demonstrate how to achieve high code coverage.

### gRPC and server
One test case for successful authentication<br />
One test case for failed authentication<br />
One test case for successful authorization<br />
One test case for failed authorization<br />
One test case for networking (RPC can be sent and received)<br />
