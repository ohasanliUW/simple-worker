FROM golang:1.17-bullseye


WORKDIR /go/src/worker

COPY . .
RUN cd certs && make && cd ..
RUN cp certs/rootCA.crt /etc/ssl/certs/
RUN make server
RUN make client
