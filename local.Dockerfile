FROM alpine

RUN apk update && apk add git && apk add ca-certificates

RUN mkdir -p /sandpiper
WORKDIR /sandpiper

COPY main /sandpiper/sandpiper

# ADD . /sandpiper
COPY sandpiper.docker.yml /sandpiper/config.yml

# ports
EXPOSE 80/tcp 443/tcp

ENTRYPOINT ["/sandpiper/sandpiper"]