FROM  golang:1.17.1 as builder
WORKDIR /tmp/tiny-golang-image
COPY . .
RUN GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o loa.exe ./cmd/loa/
