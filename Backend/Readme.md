<p align="center">
  <a href="https://go.dev/" target="_blank" rel="noopener noreferrer">
    <img src="https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/go/go-original-wordmark.svg" width="155" alt="Go Logo" />
  </a>
</p>

<h1 align="center">PT. FAS — Backend API</h1>

<p align="center">
  A Go-based backend service for the internal distribution, warehouse, finance, and decision-support system of <strong>PT. Furi Anugerah Sejahtera</strong>.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-Backend-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Backend" />
  <img src="https://img.shields.io/badge/PostgreSQL-Database-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/REST-API-334155?style=for-the-badge" alt="REST API" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Status-In%20Development-F59E0B?style=flat-square" alt="In Development" />
  <img src="https://img.shields.io/badge/Access-Internal%20Only-64748B?style=flat-square" alt="Internal Only" />
</p>

---

## Description

**PT. FAS Backend API** is an internal business application backend built with **Go (Golang)** and **PostgreSQL**. It provides the foundation for managing operational data, product inventory, customers, sales orders, distribution workflows, and financial information.

The project also supports a **Decision Support System (DSS)** inspired by research combining **K-Means clustering** and **Decision Tree** rules for distribution prioritization. Features are being developed and integrated incrementally; this README is not a claim that every planned module is production-ready.

### Technology Stack

- **Language:** Go (Golang)
- **API:** REST over HTTP (`net/http`)
- **Database:** PostgreSQL
- **Database driver/pooling:** `github.com/jackc/pgx/v5/pgxpool`
- **Configuration:** environment variables / `github.com/joho/godotenv`
- **Identifiers:** UUID (`github.com/google/uuid`, where applicable)
- **Development & deployment:** Docker and Docker Compose
- **Frontend integration:** Vue.js application (maintained separately)

## Main Modules

| Module | Purpose |
| --- | --- |
| Authentication & Authorization | Login and role-based access control (RBAC) |
| Products & Inventory | Product master data and stock-related workflows |
| Customers | Centralized customer/store records |
| Sales Orders | Order creation and operational workflow |
| Distribution | Warehouse fulfillment and distribution processes |
| Finance | Invoices, receivables, payments, and reporting integration |
| DSS | Distribution recommendations using Decision Tree rules and K-Means research |
| Audit & Security | Accountable internal activity and security controls |

> **Development note:** Module coverage varies by implementation stage. Refer to the registered routes and current source code for what is operational.

## Project Structure

The backend follows a modular layout similar to the structure below:

```text
Backend/
├── fas-server/
│   └── api/
│       └── main.go           # HTTP server and route registration
├── handlers/                 # API request handlers
├── middleware/               # Authentication and role checks
├── internal/
│   ├── config/               # Environment configuration
│   └── database/             # PostgreSQL connection and queries
├── services/
│   └── dss.go                # Decision-support business rules
├── go.mod
├── go.sum
└── README.md
```

> The diagram is a high-level guide; check the repository for the latest file layout.

## Requirements

- Go installed and available in your terminal
- PostgreSQL running locally or through Docker
- Git
- Docker Desktop + Docker Compose (optional, for containerized development)

## Project Setup

Open PowerShell in your backend directory:

```powershell
cd D:\vanmin\Backend

go mod download
```

Create your local environment file if an `.env.example` is present (if your project already has an `.env`, keep its existing values):

```powershell
Copy-Item .env.example .env
```

Example **`.env.example`** configuration for a local PostgreSQL installation:

```dotenv
PGHOST=localhost
PGPORT=5432
PGUSER=postgres
PGPASSWORD=replace_with_your_local_password
PGDATABASE=fas_db
PGSSLMODE=disable
```

**Never commit `.env`, database passwords, API keys, or real customer data to GitHub.** Put `.env` in `.gitignore` and commit only a sanitized `.env.example`.

## Compile and Run

```powershell
# Run the API with an explicit environment file
go run ./fas-server/api -env .env

# Build the backend executable
go build -o fas-api.exe ./fas-server/api

# Run the compiled application
.\fas-api.exe -env .env
```

The API is currently configured to listen at:

```text
http://localhost:8080
```

## API Endpoints

Representative endpoints from the project include:

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/health` | HTTP server health check |
| `GET` | `/ready` | PostgreSQL readiness check |
| `POST` | `/api/v1/auth/login` | User login |
| `GET` | `/api/v1/products` | Product listing |
| `GET` | `/api/v1/customers` | Customer listing |
| `GET` | `/api/v1/sales-orders` | Sales order listing |
| `POST` | `/api/v1/sales-orders` | Create a draft sales order |
| `GET` | `/api/v1/predictions` | Retrieve DSS predictions |
| `POST` | `/api/v1/predictions` | Create a DSS prediction |

> Protected endpoints require valid authentication and appropriate role permissions. Route behavior and availability should be checked against the running version.

### Health and Readiness Tests

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET

Invoke-RestMethod -Uri "http://localhost:8080/ready" -Method GET
```

### DSS Prediction Example

The following illustrates the request body. Authenticate as required by your current backend middleware before using a protected endpoint.

```json
{
  "jumlah": 100,
  "cl_qty": 50
}
```

Example request with an authenticated session token:

```powershell
$token = "YOUR_ACCESS_TOKEN"

$body = @{
    jumlah = 100
    cl_qty = 50
} | ConvertTo-Json

Invoke-RestMethod `
    -Uri "http://localhost:8080/api/v1/predictions" `
    -Method POST `
    -Headers @{ Authorization = "Bearer $token" } `
    -ContentType "application/json" `
    -Body $body
```

A prediction response can include cluster information and a distribution recommendation; exact values depend on the configured DSS rules.

## Run Tests

```powershell
# Run all available Go tests
go test ./...

# Check for common source issues
go vet ./...

# Format Go source files
go fmt ./...
```

> Test coverage depends on the test files present in the repository. Do not interpret a successful `go test` with no test files as full test coverage.

## Docker Development

When using the project's Docker Compose configuration located in the parent project directory (`D:\vanmin\docker-compose.yml`):

```powershell
cd D:\vanmin

docker compose up -d --build

docker compose ps

docker compose logs -f backend
```

For containers sharing a Compose network, the backend typically connects to the PostgreSQL service with `PGHOST=postgres`. For a backend started directly on Windows, use the appropriate local database host, typically `localhost`.

Ensure Docker Desktop is running and that the Compose environment values match the database configuration.

## Security Notes

- Keep the repository **private** for internal company code.
- Store secrets in environment variables or a secure secret manager, **not** in source files.
- Enforce authentication and RBAC **in the backend**, not only in the Vue interface.
- Validate incoming requests and uploaded files before processing them.
- Restrict CORS to approved frontend origins before production deployment.
- Use HTTPS for externally accessible services.
- Keep database backups and maintain audit logs for important operations.
- Never use real financial transactions or customer data in public demonstrations.

## Roadmap

- [ ] Complete cross-module integration for warehouse, distribution, and finance
- [ ] Expand and validate K-Means + Decision Tree DSS integration
- [ ] Customer QR portal for invoices and Surat Pesanan (SP) uploads
- [ ] Store incoming SP documents securely and show them in Reports
- [ ] Payment gateway integration and sandbox validation
- [ ] Strengthen auditing, access controls, and production hardening

## Resources

- [Go Documentation](https://go.dev/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [Docker Documentation](https://docs.docker.com/)

## Repository & Usage

**Organization:** PT. Furi Anugerah Sejahtera (PT. FAS)  
**Project:** Internal Company Management System  
**Repository:** [Vansyahlee/vanmin](https://github.com/Vansyahlee/vanmin)

This project is intended for **internal company development and operations**. No open-source license is implied. Redistribution or external access requires authorization from the repository owner.
