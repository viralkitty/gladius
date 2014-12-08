FROM golang:1.3

ENV GLADIUS_HTTP_PORT 8080
ENV DOCKER_SOCK_PATH unix:///var/run/docker.sock

EXPOSE 8080

ADD . /go/src/git.corp.adobe.com/typekit/gladius/

RUN go get -d -v ... && go install git.corp.adobe.com/typekit/gladius

CMD ["gladius"]
