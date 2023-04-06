ARG BASE=golang:1.18-alpine3.15
FROM ${BASE} AS builder

ARG ALPINE_PKG_BASE="make git gcc libc-dev libsodium-dev zeromq-dev"
ARG ALPINE_PKG_EXTRA=""

RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
RUN apk add --update --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn

ARG MAKE=make build

WORKDIR /device

COPY . .
RUN go build -o driver-box

# Next image - Copy built Go binary into new workspace
FROM alpine:3.14

RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
# dumb-init is required as security-bootstrapper uses it in the entrypoint script
RUN apk add --update --no-cache ca-certificates zeromq dumb-init curl
RUN apk --update add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata && \
    rm -rf /var/cache/apk/*

WORKDIR /
COPY --from=builder /device/driver-box /driver-box
COPY --from=builder /device/res /res
COPY --from=builder /device/scripts /scripts

EXPOSE 59999

ENTRYPOINT ["/driver-box"]
CMD ["-cp=consul.http://edgex-core-consul:8500", "--registry", "--confdir=/res"]
