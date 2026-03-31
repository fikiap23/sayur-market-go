FROM golang:1.26.1-alpine3.23

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct

COPY . .

RUN go mod download
RUN go build -o main main.go

EXPOSE 8084

CMD ["./main", "start"]