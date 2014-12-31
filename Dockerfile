FROM ubuntu:14.04

# env vars
ENV DEBIAN_FRONTEND noninteractive
ENV GLADIUS_HTTP_PORT 8080
ENV DOCKER_SOCK_PATH unix:///var/run/docker.sock
ENV GOPATH /go
ENV GOLANG_VERSION 1.3.3
ENV GOLANG_URL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz
ENV PATH /usr/src/go/bin:/go/bin:$PATH
ENV MESOS github.com/mesos/mesos-go
ENV MESOS_GIT_URL https://$MESOS.git
ENV MESOS_GO_PATH $GOPATH/src/$MESOS
ENV GLADIUS git.corp.adobe.com/typekit/gladius
ENV GLADIUS_GO_PATH $GOPATH/src/$GLADIUS

# open the port
EXPOSE 8080

# prerequisites
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gcc \
    libc6-dev \
    make \
    bzr \
    git \
    mercurial && \
    curl -ksSL $GOLANG_URL | tar -v -C /usr/src -xz && \
    cd /usr/src/go/src && \
    ./make.bash --no-clean 2>&1 && \
    git clone $MESOS_GIT_URL $MESOS_GO_PATH && \
    cd $MESOS_GO_PATH && \
    go get github.com/tools/godep && \
    godep restore && \
    cd $GOPATH && \
    go get github.com/fsouza/go-dockerclient

# copy gladius
ADD . /gladius

# install gladius
RUN mkdir -p $GLADIUS_GO_PATH && \
    cp -r /gladius $(dirname $GLADIUS_GO_PATH) && \
    cd $GOPATH && \
    go install $GLADIUS

VOLUME /gladius

CMD ["gladius"]
