FROM golang:1.17-bullseye


WORKDIR /go/src/worker

COPY . .
RUN make certs
RUN cp certs/rootCA.crt /etc/ssl/certs/
RUN make server
RUN make client
