# 基础镜像，基于golang的alpine镜像构建--编译阶段
FROM golang:alpine AS builder
COPY . /usr/local/gowork/ps-go
WORKDIR /usr/local/gowork/ps-go
RUN ls -l
ENV GOPROXY https://goproxy.cn,direct
RUN go mod tidy && go build -o /usr/local/build/ps-go


FROM scratch AS runner
WORKDIR /app/build/ps-go
COPY --from=builder /usr/local/build/ps-go .
EXPOSE 8080
ENTRYPOINT ["./main"]
