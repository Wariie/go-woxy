FROM golang:1.14.5-alpine3.12
COPY . /go-woxy
WORKDIR /go-woxy
EXPOSE 2000
EXPOSE 53
RUN go mod download
RUN go build
RUN ./go-woxy ./cfg.yml
