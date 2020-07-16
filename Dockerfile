FROM golang:1.14.5-alpine3.12
ENV GO111MODULE=on
COPY . /go-woxy
WORKDIR /go-woxy
EXPOSE 2000
EXPOSE 53
#RUN go get -u github.com/gin-gonic/gin
RUN go build
RUN ./go-woxy ./cfg.yml
