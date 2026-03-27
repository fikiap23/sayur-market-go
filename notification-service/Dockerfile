FROM golang:1.26.1-alpine3.23

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o main main.go

EXPOSE 8081

CMD ["./main", "notif-start"]