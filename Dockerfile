FROM tikhoplav/go as prepare
ENV GOPROXY direct
ENV GOSUMDB off
WORKDIR /source

COPY . .

RUN go mod tidy
RUN go build

EXPOSE 2000

RUN ["./go-woxy","./cfg.yml"] 
#ENV GO111MODULE=auto
