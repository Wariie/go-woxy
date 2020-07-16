FROM golang:1.14
ENV GO111MODULE=auto
ENV GOPROXY direct
WORKDIR /go/src/guilhem-mateo.fr/go-woxy
COPY . .
EXPOSE 80
#RUN getent hosts dl-cdn.alpinelinux.org
#RUN echo -e "http://dl-cdn.alpinelinux.org/alpine/v3.12/main\nhttp://dl-cdn.alpinelinux.org/alpine/v3.12/community" > /etc/apk/repositories
#RUN apk update && apk add git
RUN go get github.com/gin-gonic/gin
RUN go build
EXPOSE 2000
EXPOSE 53
RUN ["./go-woxy","./cfg.yml"] 
