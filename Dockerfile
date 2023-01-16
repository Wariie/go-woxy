FROM golang:1.19.5-alpine as prepare
ENV GOPROXY direct
ENV GOSUMDB off
WORKDIR /source
COPY . .
RUN go mod tidy
RUN go build
ENTRYPOINT ["./go-woxy","./cfg.yml"] 
EXPOSE 2000/tcp
#ENV GO111MODULE=auto
