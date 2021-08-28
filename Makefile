all: grpc server client
grpc:
	protoc --go_out=. --go_opt=paths=source_relative \
    		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    		protobuf/worker.proto

server:
	go build srv/server.go

client:
	go build clnt/client.go

clean:
	rm client server protobuf/*.pb.go

test:
	go test srv/*.go --count=1
	go test job/*go --count=1