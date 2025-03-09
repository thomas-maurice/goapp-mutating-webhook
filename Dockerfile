FROM golang:alpine

COPY . /go
RUN apk add --update ca-certificates
RUN go get && go build -o mutating-webhook

FROM scratch
COPY --from=0 /etc/ssl /etc/ssl
COPY --from=0 /etc/ca-certificates /etc/ca-certificates
COPY --from=0 /go/mutating-webhook /mutating-webhook
