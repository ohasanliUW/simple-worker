all: grpc server client certs

certs:
	$(MAKE) -C certs
grpc:
	protoc --go_out=. --go_opt=paths=source_relative \
    		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    		protobuf/worker.proto

server:
	go build srv/server.go

client:
	go build clnt/client.go

clean:
	rm client server ||:
	rm protobuf/*.pb.go ||:
	$(MAKE) -C certs clean

test: grpc
	go test srv/*.go --count=1
	go test job/*go --count=1

.PHONY: all certs grpc
