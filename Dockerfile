# builder
FROM golang:alpine as builder

RUN apk update && apk add git && apk add ca-certificates

RUN go get github.com/gabstv/sandpiper/sandpiper
WORKDIR $GOPATH/src/github.com/gabstv/sandpiper/sandpiper
RUN go build -o /go/bin/sandpiper

# buid
FROM alpine

#RUN mkdir -p /sandpiper # mkdir not available on scratch
WORKDIR /sandpiper

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/sandpiper /sandpiper/sandpiper

# ADD . /sandpiper
COPY sandpiper.docker.yml /sandpiper/config.yml

# ports
EXPOSE 80/tcp 443/tcp

ENTRYPOINT ["/sandpiper/sandpiper"]