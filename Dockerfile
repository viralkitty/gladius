FROM ubuntu:14.04

ENV DEBIAN_FRONTEND noninteractive

ENV GLADIUS_HTTP_PORT 8080
ENV DOCKER_SOCK_PATH unix:///var/run/docker.sock

ENV GOPATH /go
ENV PATH /go/bin:$PATH

RUN apt-get update && apt-get install -y \
        ca-certificates curl gcc libc6-dev make \
        bzr git mercurial \
        --no-install-recommends

ENV GOLANG_VERSION 1.3.3

RUN curl -k -sSL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz \
        | tar -v -C /usr/src -xz

RUN cd /usr/src/go/src && ./make.bash --no-clean 2>&1

ENV PATH /usr/src/go/bin:$PATH

ADD . /go/src/git.corp.adobe.com/typekit/gladius/

RUN mkdir -p /go/src/github.com/mesos && \
    cd /go/src/github.com/mesos && \
    git clone https://github.com/mesos/mesos-go.git /go/src/github.com/mesos/mesos-go && \
    cd /go/src/github.com/mesos/mesos-go && \
    go get github.com/tools/godep && \
    godep restore && \
    cd examples && \
    go build -tags=test-exec -o test-executor test_executor.go

RUN cd /go && go get -d -v ... && go install git.corp.adobe.com/typekit/gladius

EXPOSE 8080
CMD ["gladius"]
