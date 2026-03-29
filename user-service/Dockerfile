FROM golang:1.26.1-alpine3.23

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main main.go

EXPOSE 8090

CMD ["./main", "start"]
