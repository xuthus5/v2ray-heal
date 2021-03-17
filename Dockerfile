FROM golang:alpine
WORKDIR /src
ADD . /src/
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
apk add --update --no-cache gcc bash musl-dev && GO111MODULE=on GOPROXY=https://goproxy.cn GOOS=linux CGO_ENABLED=1 \
GOARCH=amd64 go build -ldflags="-s -w" -o cli .
EXPOSE 2131
ENTRYPOINT ["./docker-entrypoint.sh"]
