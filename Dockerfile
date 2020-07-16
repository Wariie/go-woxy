FROM golang:1.14.5-alpine3.12
ENV GO111MODULE=auto
ENV GOPROXY direct
WORKDIR /go/src/guilhem-mateo.fr/go-woxy
COPY . .
EXPOSE 80
RUN echo http://dl-6.alpinelinux.org/alpine/v3.12/main > /etc/apk/repositories; \
    echo http://mirror.yandex.ru/mirrors/alpine/v3.12/community >> /etc/apk/repositories
RUN apk update -qq && apk add git
RUN go get github.com/gin-gonic/gin
RUN go build
EXPOSE 2000
EXPOSE 53
RUN ["./go-woxy","./cfg.yml"] 
