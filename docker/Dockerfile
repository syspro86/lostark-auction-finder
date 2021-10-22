FROM  golang:1.16 as builder
WORKDIR /tmp/tiny-golang-image
COPY . .
RUN GOOS=windows GOARCH=amd64 go build -o loa.exe cmd/loa/*.go
