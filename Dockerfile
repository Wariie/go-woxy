FROM tikhoplav/go as prepare
ENV GOPROXY direct
ENV GOSUMDB off
WORKDIR /source

COPY . .

RUN go mod tidy
RUN go build

CMD ["./go-woxy","./cfg.yml"] 
EXPOSE 2000/tcp
#ENV GO111MODULE=auto
