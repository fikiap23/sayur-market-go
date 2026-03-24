## 📦 Database Migration Guide

### 🔧 Environment Variables

```bash
export DATABASE_HOST=localhost
export DATABASE_PORT=5432
export DATABASE_USER=postgres
export DATABASE_PASSWORD=lokal
export DATABASE_NAME=user_service
```

### 🔗 PostgreSQL Connection URL

```bash
export POSTGRESQL_URL="postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}?sslmode=disable"
```

---

## 🚀 Migration Commands

> Jalankan dari root folder: `user-service`

### ▶️ Run Migration (Up)

```bash
migrate -database ${POSTGRESQL_URL} -path database/migrations up
```

### ⏪ Rollback Migration (Down)

```bash
migrate -database ${POSTGRESQL_URL} -path database/migrations down
```

### 🔄 Force Version (Jika dirty)

```bash
migrate -database ${POSTGRESQL_URL} -path database/migrations force <version>
```

---

## 📝 Create Migration File

```bash
migrate create -ext sql -dir database/migrations -seq create_users_table
```

---

## 🐳 Alternative (Docker)

### ▶️ Run Migration

```bash
docker run -v $(pwd)/database/migrations:/migrations --network host --rm migrate/migrate \
  -database "${POSTGRESQL_URL}" -path /migrations up
```

### 📝 Create Migration

```bash
docker run -v $(pwd)/database/migrations:/migrations --rm migrate/migrate \
  create -ext sql -dir /migrations -seq create_users_table
```

---

## ⚠️ Notes

- Jalankan command dari folder `user-service`
- Pastikan database sudah running
- Gunakan `sslmode=disable` untuk local
- Gunakan `force` jika migration dirty
- Jalankan migration setiap update schema

---
