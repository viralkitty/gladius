FROM golang:1.3

COPY . /go/src/git.corp.adobe.com/typekit/gladius/

RUN go get -d -v ... && go install git.corp.adobe.com/typekit/gladius

CMD ["gladius"]
