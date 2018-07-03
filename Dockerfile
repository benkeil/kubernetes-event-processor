FROM golang:1.10.3-alpine3.7 AS builder

RUN apk add --update git

WORKDIR /go/src/build

COPY . .

RUN go get -u github.com/golang/dep/cmd/dep && \
    dep ensure && \
    go build -o app


FROM alpine:3.7

WORKDIR /var/opt/

COPY --from=builder /go/src/build/app ./kubernetes-event-processor

COPY config.yaml .

ENTRYPOINT ./kubernetes-event-processor
