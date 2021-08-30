# simple-worker
Teleport Code Challenge

### Build
```
# build
make all
```
The above will build everything you need to run the server and client

### Run server
Simply:
```
./server
```

### Run client
Check available options
```
./client --help
```
`testdata/echo.sh` is a simple countdown script. It takes an integer as command line argument and counts down from that number to 1 by sleeping 1 second between each iteration. It also prints the current number to both stdout and stderr.
To test output streaming via RPC, start a job that runs this script. 
```
# call client to execute the script @ testdata/echo.sh
./client start -c "./testdata/echo.sh 120"
```
Once the job starts, use printed job id to stream output.
NOTE: Multiple clients can stream simultanously as long as certificates being used is associated with the same username (embedded into CN  field); try output command below in two different terminal windows.
```
./client output -j <job-id>
```

### Run go tests
There are two suites of tests
1. Library
2. Server

Run
```
go test -race job/*.go -v --count=1
go test -race srv/*.go -v --count=1
```
