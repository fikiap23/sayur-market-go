# Micro Sayur Service

Micro Sayur Service adalah aplikasi microservice untuk manajemen toko sayur online.

## Prasyarat

Sebelum memulai, pastikan sistem Anda telah memiliki:

- Go 1.19 atau lebih tinggi
- PostgreSQL
- RabbitMQ
- Redis
- Elasticsearch
- Docker & Docker Compose (opsional)

## Konfigurasi Environment

Salin file `.env.example` ke `.env` dan sesuaikan dengan konfigurasi lokal Anda:

```bash
cp .env.example .env
```

Jalankan go mod lewat makefile:
```bash
make mod-tidy

make mod-download

make mod-all
```