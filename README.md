
<div align="center">

<img src="Frontend/src/assets/fas-ai/fas-ai-logo.png" alt="PT FAS Logo" width="200">

# PT. FURI ANUGERAH SEJAHTERA

### Enterprise Distribution & Business Management System

Sistem manajemen distribusi, pergudangan, keuangan, dan pengambilan keputusan berbasis data.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Vue.js](https://img.shields.io/badge/Vue.js-3-4FC08D?style=for-the-badge&logo=vuedotjs&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)

</div>

---

## About PT. FAS

PT. FAS merupakan aplikasi manajemen internal perusahaan yang mengintegrasikan seluruh proses operasional ke dalam satu sistem.

Sistem dirancang menggunakan backend Go, frontend Vue 3, dan database PostgreSQL.

## Technology Stack

| Component | Technology |
|-----------|------------|
| Backend | Go (Golang) |
| API | REST API |
| Frontend | Vue 3 |
| Build Tool | Vite |
| Styling | Tailwind CSS |
| Database | PostgreSQL |
| Infrastructure | Docker Compose |
| Authentication | Role-Based Access Control |
| Decision Support | K-Means & Decision Tree |

## Main Modules

- Overview Dashboard
- Inventory & Stock Management
- Sales Order & Operations
- Warehouse Management
- Distribution Management
- Finance & Accounting
- Invoice & Accounts Receivable
- Payment Management
- Tax & Financial Reporting
- Finance Analytics
- FAS Intelligence
- User Management & Access Control

## System Architecture

Frontend (Vue 3 + Vite)

↓

Backend REST API (Go)

↓

PostgreSQL Database

The backend handles business logic, authentication, financial processing, warehouse operations, and decision-support services.

## Project Structure

    vanmin/
    ├── Backend/
    │   ├── fas-server/
    │   │   └── api/
    │   ├── handlers/
    │   ├── internal/
    │   │   ├── config/
    │   │   └── database/
    │   ├── middleware/
    │   ├── services/
    │   ├── go.mod
    │   └── go.sum
    │
    ├── Frontend/
    │   ├── public/
    │   ├── src/
    │   │   ├── assets/
    │   │   ├── components/
    │   │   ├── router/
    │   │   ├── services/
    │   │   ├── stores/
    │   │   └── views/
    │   ├── package.json
    │   └── vite.config.js
    │
    ├── docker-compose.yml
    ├── .gitignore
    └── README.md

## Backend Development

Requirements:

- Go 1.26.5
- PostgreSQL

From the Backend directory:

    go test ./...

    go run ./fas-server/api

Configure the required environment variables locally before running the application.

## Frontend Development

From the Frontend directory:

    npm install

    npm run dev

Production build:

    npm run build

## Security

This project is intended for internal business operations.

- Keep credentials and API keys in environment variables.
- Do not commit .env files containing secrets.
- Restrict access to company financial and operational data.
- Apply role-based authorization to protected endpoints.

---

<div align="center">

**PT. FURI ANUGERAH SEJAHTERA**

Built with Go, Vue 3, and PostgreSQL.

</div>
