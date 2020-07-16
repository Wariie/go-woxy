#FROM tikhoplav/go
FROM golang:1.13.1-alpine3.10 as prepare
ENV GOPROXY direct
WORKDIR /source

COPY go.mod .
COPY go.sum .

RUN go mod download

#ENV GO111MODULE=auto

#WORKDIR /go/src/guilhem-mateo.fr/go-woxy
#COPY . .
#RUN go get github.com/gin-gonic/gin
#RUN go build
#EXPOSE 2000
#EXPOSE 53
#RUN ["./go-woxy","./cfg.yml"] 
