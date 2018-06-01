FROM golang:latest
MAINTAINER Gabriel Ochsenhofer <gabriel.ochsenhofer@gmail.com>

RUN mkdir -p /sandpiper
WORKDIR /sandpiper

# ADD . /sandpiper
COPY sandpiper.docker.yml /sandpiper/config.yml
RUN go get github.com/gabstv/sandpiper/sandpiper

# ports
EXPOSE 80/tcp 443/tcp

ENTRYPOINT ["sandpiper"]