FROM golang:1.18-alpine AS builder

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY ./config ./config
COPY ./core ./core
COPY ./driver ./driver
COPY ./driver-config ./driver-config
COPY ./res ./res
COPY ./go.sum ./go.sum
COPY ./go.mod ./go.mod
COPY ./main.go ./main.go

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && \
    apk update && apk add pkgconfig zeromq-dev gcc libc-dev && \
    go mod vendor && \
    go build -o driver-box .

FROM alpine:latest

WORKDIR /

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && \
    apk update && apk add zeromq-dev curl tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata && \
    rm -rf /var/cache/apk/*

COPY --from=builder /build/app /app
COPY --from=builder /build/res /res
COPY --from=builder /build/driver-box /driver-box

EXPOSE 59999

ENTRYPOINT ["/driver-box"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "--confdir=/res"]