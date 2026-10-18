
<div align="center">

<img src="Frontend/src/assets/fas-ai/fas-ai-logo.png" alt="FAS Logo" width="280">

# FAS

### Full-Stack Application

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Vue.js](https://img.shields.io/badge/Vue.js-3-4FC08D?style=for-the-badge&logo=vuedotjs&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)

</div>

---

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

## System Architecture

```text
     Vue 3 + Vite
           |
           v
     Go REST API
           |
           v
      PostgreSQL
```

The backend handles application logic, authentication, data processing, and decision-support services.

---

## Development

### Backend Development

Requirements:

- Go 1.26.5
- PostgreSQL

Run the backend:

```bash
cd Backend

go test ./...

go run ./fas-server/api
```

Configure the required environment variables locally before starting the application.

### Frontend Development

Install dependencies:

```bash
cd Frontend

npm install
```

Start the development server:

```bash
npm run dev
```

Build for production:

```bash
npm run build
```

---

## Security

- Store credentials and API keys in environment variables.
- Never commit sensitive `.env` files.
- Use role-based access control.
- Protect private application data.
- Validate and test changes before deployment.

---

<div align="center">

<sub>Built with Go, Vue.js, PostgreSQL & Docker.</sub>

</div>
