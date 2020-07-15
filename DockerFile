FROM golang:latest
COPY . /go-woxy
WORKDIR /go-woxy
RUN go build
RUN ./go-woxy ./cfg.yml
EXPOSE 2000