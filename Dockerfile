FROM golang:latest
COPY . /go-woxy
WORKDIR /go-woxy
EXPOSE 2000
EXPOSE 53
RUN go build
RUN ./go-woxy ./cfg.yml
