# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

RUN mkdir -p /app
WORKDIR /app
ADD . /app

RUN mkdir -p /go/src/github.com/verticallabs
RUN ln -s /app /go/src/github.com/verticallabs/urlsafety
RUN go get -d ./...

# Document that the service listens on port 8080.
EXPOSE 8080
