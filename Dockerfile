FROM golang:alpine

COPY . /go
RUN cd /go
RUN apk add --update ca-certificates
RUN go build -o /go/mutating-webhook ./example

FROM scratch
COPY --from=0 /etc/ssl /etc/ssl
COPY --from=0 /etc/ca-certificates /etc/ca-certificates
COPY --from=0 /go/mutating-webhook /mutating-webhook
