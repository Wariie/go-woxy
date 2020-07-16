FROM golang:alpine3.12
ENV GO111MODULE=auto
RUN export GOPATH=/go
ENV GOPATH /go
ENV GOPROXY direct
COPY . /go/src/guilhem-mateo.fr/go-woxy
WORKDIR /go/src/guilhem-mateo.fr/go-woxy
RUN export PATH=$PATH:$GOPATH/bin
ENV PATH $PATH:$GOPATH/bin
EXPOSE 2000
EXPOSE 53
RUN go get github.com/gin-gonic/gin
RUN go build
RUN ["./go-woxy","./cfg.yml"] 
