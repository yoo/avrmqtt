FROM golang:alpine AS builder

COPY . /go/src/github.com/yoo/avrmqtt
WORKDIR /go/src/github.com/yoo/avrmqtt

RUN set -x \
 && go install -mod=vendor github.com/yoo/avrmqtt

FROM alpine:latest

COPY --from=builder /go/bin/avrmqtt /usr/bin

RUN set -x \
 && apk add --update --no-cache dumb-init ca-certificates \
 && adduser -D avrmqtt
 
USER avrmqtt

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/bin/avrmqtt"]
