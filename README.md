# url-shortener

A simple URL shortener API built in Go, with a Next.js frontend, SQLite persistence, Docker support, and Kubernetes deployment using k0s.

---

## Features

- Go backend API
- Next.js frontend
- SQLite database
- /live and /health endpoints for Kubernetes
- Docker support
- Kubernetes (StatefulSet + PVC + Service)

---

## Local development

### Backend

```Bash

sudo mkdir -p /data
sudo chown -R $USER:$USER /data
go run .
```
Backend runs on:
```
http://localhost:8081
```
---

### Frontend

```Bash

cd frontend
cp .env.local.example .env.local
npm install
npm run dev
```
Frontend runs on:
```
http://localhost:3000
```
---

## Environment variables

Create `.env.local` in frontend:

```env

NEXT_PUBLIC_API_URL=http://localhost:8081
```
If not set, frontend automatically falls back to:
```
http://localhost:8081
```
---

## API Testing

Health checks:

```Bash

curl http://localhost:8081/live
curl http://localhost:8081/health
```
Create short URL:

```Bash

curl -X POST http://localhost:8081 \
  -H "Content-Type: application/json" \
  -d '{"url":"https://google.com"}'
```
---

## SQLite database

Database location:
```
/data/urls.db
```
Check content:

```Bash

sqlite3 /data/urls.db "SELECT * FROM urls;"
```
---

## Docker

Build image:

```Bash

docker build -t url-shortener:dev .
```
---

## Kubernetes (k0s)

### Import image

```Bash

docker save url-shortener:dev -o url-shortener.tar
sudo k0s ctr images import url-shortener.tar
```
### Deploy

```Bash

sudo k0s kubectl apply -k k8s/base
```
### Verify

```Bash

sudo k0s kubectl get pods
sudo k0s kubectl get svc
sudo k0s kubectl get pvc
```
---

## Access service

```Bash

curl http://localhost:30080/live
curl http://localhost:30080/health
```
---

## Architecture

- Backend: Go API (port 8081)
- Frontend: Next.js
- Storage: SQLite (mounted `at /data`)
- Kubernetes:
  - StatefulSet (persistent pod)
  - PVC (database storage)
  - NodePort service (30080)

---

## Notes

- Backend port changed from 8080 → 8081 to avoid conflict with k0s (kube-router)
- Dockerfile, Kubernetes manifests, and tests updated accordingly
- Frontend supports automatic fallback if `.env.local` is missing
- Database is persisted via Kubernetes PVC

---
