# simple-worker
Teleport Code Challenge

### Build
Build will generate protobuf code via `protoc` then start building a docker image for running services in Linux env.
```
# build
make all
# server will be started automatically by Makefile
```

### Run client
```
# call client to execute a script @ testdata/echo.sh
docker exec -it worker ./client start -c "./testdata/echo.sh 1200"
```
Once the job starts, use printed job id to stream output.
NOTE: Multiple clients can stream simultanously as long as certificates being used is associated with the same username (embedded into CN  field); try output command below in two different terminal windows.
```
docker exec -it worker ./client output -j <job-id>
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
