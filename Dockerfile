FROM golang:1.21-alpine as builder  
WORKDIR /app  
ENV GOPROXY=https://goproxy.cn  
COPY ./go.mod /app
COPY ./go.sum /app
COPY ./main.go /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o cdk-web-service  

FROM busybox as runner
COPY --from=builder /app/cdk-web-service /app
ENTRYPOINT ["/app"]
