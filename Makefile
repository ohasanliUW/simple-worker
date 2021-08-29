all: grpc docker

certs:
	$(MAKE) -C certs all
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
	docker stop worker
	docker rm worker

test: grpc
	go test srv/*.go --count=1
	go test job/*go --count=1


docker:
	sudo docker build -t worker .
	docker run -d --name worker worker ./server
