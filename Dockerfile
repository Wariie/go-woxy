FROM golang:1.14.5-alpine3.12
ENV GO111MODULE=auto
ENV GOPROXY direct
WORKDIR /go/src/guilhem-mateo.fr/go-woxy
COPY . .
RUN curl -v -s http://dl-cdn.alpinelinux.org/alpine/edge/main/x86_64/APKINDEX.tar.gz > /dev/null
RUN apk update -qq && apk add git
RUN go get github.com/gin-gonic/gin
RUN go build
EXPOSE 2000
EXPOSE 53
RUN ["./go-woxy","./cfg.yml"] 
