# 使用官方的 golang 基础镜像
FROM golang:1.20.5

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件
COPY go.mod ./

# 复制所有文件到容器中
COPY . .

# 构建应用程序
RUN go build -o myapp

# 暴露端口
EXPOSE 8080

# 启动应用程序
CMD ["./myapp"]

