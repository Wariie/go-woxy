#FROM tikhoplav/go
FROM tikhoplav/go as prepare
ENV GOPROXY direct
WORKDIR /source

COPY . .

RUN go mod download

#ENV GO111MODULE=auto

#WORKDIR /go/src/guilhem-mateo.fr/go-woxy
#COPY . .
#RUN go get github.com/gin-gonic/gin
#RUN go build
#EXPOSE 2000
#EXPOSE 53
#RUN ["./go-woxy","./cfg.yml"] 
