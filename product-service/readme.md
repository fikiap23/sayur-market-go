# 📦 Product Service — Database Migration Guide

## 🔧 Environment Setup

Gunakan file `.env` di root project:

```env
POSTGRESQL_URL=postgres://postgres:lokal@product_db:5432/product_service?sslmode=disable
```

---

## 🚀 Load Environment Variables

Sebelum menjalankan migration, load `.env`:

```bash
source .env
```

> Jalankan ini setiap buka terminal baru

---

## 🚀 Migration Commands

> Jalankan dari root folder: `product-service`

### ▶️ Run Migration (Up)

```bash
migrate -database "$POSTGRESQL_URL" -path database/migrations up
```

---

### ⏪ Rollback Migration (Down)

```bash
migrate -database "$POSTGRESQL_URL" -path database/migrations down
```

---

### 🔄 Force Version (Jika Dirty State)

```bash
migrate -database "$POSTGRESQL_URL" -path database/migrations force <version>
```

---

## 📝 Create Migration File

```bash
migrate create -ext sql -dir database/migrations -seq create_products_table
```

---

## 🐳 Alternative (Docker)

> Pastikan `.env` sudah di-load dengan `source .env`

### ▶️ Run Migration

```bash
docker run -v $(pwd)/database/migrations:/migrations \
  --network host \
  -e POSTGRESQL_URL \
  --rm migrate/migrate \
  -database "$POSTGRESQL_URL" -path /migrations up
```

---

### 📝 Create Migration

```bash
docker run -v $(pwd)/database/migrations:/migrations \
  --rm migrate/migrate \
  create -ext sql -dir /migrations -seq create_products_table
```

---

## ⚙️ Optional (Recommended) — Makefile

Buat file `Makefile` di root project:

```makefile
include .env

migrate-up:
	migrate -database "$(POSTGRESQL_URL)" -path database/migrations up

migrate-down:
	migrate -database "$(POSTGRESQL_URL)" -path database/migrations down

migrate-force:
	migrate -database "$(POSTGRESQL_URL)" -path database/migrations force $(v)

migrate-create:
	migrate create -ext sql -dir database/migrations -seq $(name)
```

### ▶️ Usage

```bash
make migrate-up
make migrate-down
make migrate-force v=1
make migrate-create name=create_products_table
```

---

## ⚠️ Notes

- Jalankan command dari folder `product-service`
- Gunakan `source .env` (hindari `export` manual)
- Pastikan database sudah running
- Gunakan `force` jika migration dalam kondisi dirty
- Jalankan migration setiap ada perubahan schema

---

## 📁 Recommended Structure

```
product-service/
├── .env
├── Makefile
├── database/
│   └── migrations/
├── main.go
```

---
