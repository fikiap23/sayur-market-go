FROM golang:1.26.1-alpine3.23

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct

COPY . .

RUN go mod download
RUN go build -o main main.go
# RUN go build -o worker-order cmd/worker-order.go
# RUN go build -o worker-update-payment-order cmd/worker-update-payment-order.go

EXPOSE 8083

CMD ["./main", "start"]