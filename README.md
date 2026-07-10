# Skill Arena

Skill Arena is a competitive gaming platform backend and frontend workspace.

## Backend

The backend is a Go service located in `backend/`.

Common verification commands:

```powershell
go test ./...
go vet ./...
go build ./...
gofmt -w .
```

## Frontend

The frontend is a Next.js application located in `frontend/`.

## Runtime Data

Development JSON data is kept under `backend/data/`. Production deployments should use the configured production services for database, cache, storage, payments, email and observability.
