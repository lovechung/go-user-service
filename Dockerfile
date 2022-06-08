FROM golang:1.18 AS builder

COPY . /src
WORKDIR /src

ENV GOPROXY https://goproxy.cn,direct
# 编译，关闭CGO，防止编译后的文件有动态链接，而alpine镜像里有些c库没有，直接没有文件的错误
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 make build

FROM alpine

RUN echo "https://mirrors.aliyun.com/alpine/v3.8/main/" > /etc/apk/repositories \
    && echo "https://mirrors.aliyun.com/alpine/v3.8/community/" >> /etc/apk/repositories \
    && apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime  \
    && echo Asia/Shanghai > /etc/timezone \
    && apk del tzdata

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 9000

CMD ["sh", "-c", "./user-service -env $ENV -config_type $CONFIG_TYPE -config_host $CONFIG_HOST -config_token $CONFIG_TOKEN"]
