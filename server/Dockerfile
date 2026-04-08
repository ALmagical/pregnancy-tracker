# 构建阶段
FROM golang:1.23-alpine AS build
WORKDIR /src
# 国内网络下默认代理易超时，构建时使用国内模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# 运行阶段
FROM alpine:3.20
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
	&& apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server ./server
ENV LOCAL_UPLOAD_DIR=/data/uploads
RUN mkdir -p /data/uploads
EXPOSE 8080
CMD ["./server"]
