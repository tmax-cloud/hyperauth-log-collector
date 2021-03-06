FROM golang:1.15

WORKDIR /go/src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o ./build/main .
RUN chmod +777 ./build/main
RUN chmod +777 start.sh
ENTRYPOINT ["/go/src/start.sh"]
