FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git upx binutils

WORKDIR /go/src/github.com/nmarcetic/docker-nginx-reload
COPY . ./
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o /go/bin/docker-nginx-reload
RUN strip /go/bin/docker-nginx-reload
RUN upx -v --ultra-brute /go/bin/docker-nginx-reload

############################

FROM  golang:alpine
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/docker-nginx-reload /docker-nginx-reload
ENTRYPOINT ["/docker-nginx-reload"]