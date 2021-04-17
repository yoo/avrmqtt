FROM golang:alpine AS builder

COPY . /avrmqtt
WORKDIR /avrmqtt

RUN set -ex; \
apk add --update --no-cache dumb-init ca-certificates; \
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags "-static"' .

RUN set -ex; \
adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \
    --uid 1000 \
    avrmqtt;

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl /etc/ssl
COPY --from=builder /avrmqtt/avrmqtt /usr/bin/avrmqtt

USER avrmqtt

CMD ["/usr/bin/avrmqtt"]
