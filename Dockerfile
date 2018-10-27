FROM golang:alpine AS builder

COPY . /go/src/github.com/JohannWeging/avrmqtt
WORKDIR /go/src/github.com/JohannWeging/avrmqtt

RUN set -x \
 && go install github.com/JohannWeging/avrmqtt

FROM johannweging/base-alpine:latest

COPY --from=builder /go/bin/avrmqtt /usr/bin

RUN set -x \
 && createuser avrmqtt
 
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["gosu", "avrmqtt", "avrmqtt"]

