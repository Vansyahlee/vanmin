<p align="center">
  <a href="https://vuejs.org/" target="_blank" rel="noopener noreferrer">
    <img src="https://cdn.jsdelivr.net/gh/devicons/devicon@latest/icons/vuejs/vuejs-original.svg" width="125" alt="Vue.js Logo" />
  </a>
</p>

<h1 align="center">PT. FAS — Frontend</h1>

<p align="center">
  A modern, responsive web interface for the internal distribution, warehouse, finance, and decision-support system of <strong>PT. Furi Anugerah Sejahtera</strong>.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Vue.js-Frontend-42B883?style=for-the-badge&logo=vuedotjs&logoColor=white" alt="Vue.js Frontend" />
  <img src="https://img.shields.io/badge/Vite-Build%20Tool-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite" />
  <img src="https://img.shields.io/badge/Tailwind%20CSS-Styling-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white" alt="Tailwind CSS" />
  <img src="https://img.shields.io/badge/JavaScript-ES6%2B-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black" alt="JavaScript" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Status-In%20Development-F59E0B?style=flat-square" alt="In Development" />
  <img src="https://img.shields.io/badge/Access-Internal%20Dashboard-64748B?style=flat-square" alt="Internal Dashboard" />
  <img src="https://img.shields.io/badge/API-Go%20%2B%20PostgreSQL-00ADD8?style=flat-square" alt="Go and PostgreSQL API" />
</p>

---

## Description

**PT. FAS Frontend** is a single-page web application built with **Vue.js**, **Vite**, and **Tailwind CSS**. It provides the user interface for managing day-to-day operations at PT. Furi Anugerah Sejahtera, including warehouse activities, distribution, sales orders, financial monitoring, and decision-support insights.

This project is developed alongside a **Go (Golang) REST API** with **PostgreSQL** as its database. The frontend focuses on accessibility, clear navigation, responsive layouts, consistent visual language, and role-appropriate business workflows.

> **Development status:** The application is under active development. The modules and roadmap below describe both existing UI areas and planned integrations; they do **not** imply every function is fully connected to live backend data or ready for production.

### Technology Stack

| Technology | Purpose |
| --- | --- |
| [Vue.js](https://vuejs.org/) | Reactive frontend components and views |
| [Vite](https://vite.dev/) | Development server and production build |
| [Vue Router](https://router.vuejs.org/) | Client-side navigation and page routing |
| [Tailwind CSS](https://tailwindcss.com/) | Utility-first styling |
| JavaScript | Application state, event handling, and API integration |
| Go REST API | Backend authentication and business services |
| PostgreSQL | Persistent business data, accessed through the backend |
| Docker / Docker Compose | Optional containerized development and deployment |

## Main Modules

| Module | Purpose |
| --- | --- |
| Landing & Login | Public landing page and login to the internal system |
| Overview Dashboard | High-level company performance metrics for authorized users |
| Operations | Sales order workflow and distribution preparation |
| Warehouse & Inventory | Product stock visibility and warehouse operations |
| Distribution | Picking, packing, shipping, and fulfillment tracking |
| Finance | Sales, invoices, receivables, payments, cash, tax, analytics, and reports |
| Decision Support System (DSS) | Distribution recommendations informed by K-Means and Decision Tree research |
| Reports | Operational and financial summaries, printing, and export workflows |
| Users & Settings | User management, profiles, company settings, and interface preferences |

### Intended User Roles

- **Admin / Owner:** company-wide administration and business overview.
- **Warehouse Staff (`pegawai`):** operational, inventory, and fulfillment activities.
- **Finance (`finance`):** financial records, reporting, payments, and customer-related finance workflows.

The exact routes available to each role are defined in the current application router. **Frontend route protection is a usability layer, not a substitute for backend authentication and authorization.**

## User Interface & Design System

- Responsive pages for desktop and mobile form factors.
- Light, Dark, and System appearance preferences.
- Accent presets: **Blue**, **Indigo**, and **Emerald**.
- Shared CSS tokens such as `--fas-accent`, `--fas-accent-text`, `--fas-accent-soft`, and `--fas-accent-border`.
- Consistent forms, action icons, dropdown menus, status indicators, and navigation patterns.
- Print/export workflows designed for business use, including finance reports and invoices.
- Target invoice layout: **9.5 × 5.5 inches**, with a maximum of **10 line items per printed page** and consistent invoice identity on continuation pages.

## Project Structure

A simplified overview of the application's organization:

```text
Frontend/
├── public/                       # Public assets (if present)
├── src/
│   ├── components/
│   │   └── layout/               # Shared dashboard layout
│   ├── router/
│   │   └── index.js              # Public and role-protected routes
│   ├── views/
│   │   ├── auth/                 # Authentication pages
│   │   ├── dashboard/            # Owner/admin overview
│   │   ├── warehouse/            # Warehouse dashboard
│   │   ├── operations/           # Sales orders and operations
│   │   ├── inventory/            # Stock views
│   │   ├── distribution/         # Distribution workflow
│   │   ├── intelligence/         # DSS views
│   │   ├── finance/              # Finance pages and reports
│   │   ├── reports/              # Reports module
│   │   ├── users/                # User management
│   │   └── settings/             # Settings and company profile
│   └── ...                       # Other application files
├── package.json
├── package-lock.json             # If npm is the project's package manager
├── index.html
└── README.md
```

> This tree is illustrative. Follow your actual `src/` structure rather than moving or replacing existing files to match the diagram.

## Requirements

- A recent supported Node.js LTS release (compatible with your installed Vite version).
- npm (or the package manager used by this repository).
- Git.
- Running Go backend and PostgreSQL for features that require live API data.
- Docker Desktop + Docker Compose (optional).

## Project Setup

Open **PowerShell** in the frontend directory:

```powershell
cd D:\vanmin\Frontend

# Install dependencies
npm install
```

If a committed `package-lock.json` is available and you want a reproducible install, use `npm ci` instead of `npm install`.

### Environment Configuration

Follow the actual API configuration used by your source code. If your project supports a Vite environment variable for the API origin, an **example** `.env.local` might be:

```dotenv
# Example only: use this name only if your frontend code reads it.
VITE_API_BASE_URL=http://localhost:8080
```

> **Important:** This is an example, not a verified environment variable in the repository. Vite exposes all `VITE_` variables to client-side code. **Never place database passwords, private API tokens, Midtrans Server Keys, or other secrets in frontend environment variables.**

## Compile and Run the Project

```powershell
cd D:\vanmin\Frontend

# Development server
npm run dev

# Make a production build
npm run build

# Preview a production build locally
npm run preview
```

Vite commonly uses `http://localhost:5173` for local development; check the URL printed by the terminal, because the actual port may differ.

### Test the Frontend on a Phone (Local Network)

For testing the mobile layout and scanning a QR code displayed on your laptop, run the frontend on a trusted local network:

```powershell
npm run dev -- --host 0.0.0.0
```

Then open the **Network** URL shown by Vite in the phone's browser (both devices must be able to reach each other). Do not open `localhost` on the phone to reach your laptop; `localhost` on the phone points to the phone itself.

If the mobile page calls the Go API, the API must also be reachable through a properly configured, secure network route. Avoid permanently exposing development ports or sensitive company services to the public internet.

> This test covers navigation and the frontend UI. A full QR invoice → customer portal → Surat Pesanan upload or Midtrans Sandbox payment flow requires its corresponding backend endpoints and proper HTTPS/payment configuration; these are separate integration milestones.

## Application Routes

Representative routes from the project structure:

| Route | Page | Audience |
| --- | --- | --- |
| `/` | Public landing page | Public |
| `/login` | Login | Public |
| `/dashboard` | Business overview | Admin |
| `/warehouse` | Warehouse dashboard | Admin / Warehouse |
| `/operations` | Operations | Admin / Warehouse |
| `/inventory` | Inventory | Admin / Warehouse |
| `/distribution` | Distribution | Admin / Warehouse |
| `/intelligence` | DSS | Authorized internal users |
| `/finance` | Finance overview | Admin / Finance |
| `/finance/analytics` | Finance analytics | Admin / Finance |
| `/finance/reports` | Finance reports | Admin / Finance |
| `/reports` | Reports | Admin / Finance |
| `/settings` | Settings | Authorized internal users |

Routes, permissions, and display names may change as the system evolves. The backend must independently verify all protected actions and data access.

## Backend Integration

The frontend communicates with the Go REST API through HTTP requests. Typical business endpoints include:

| Method | Endpoint | Function |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | Sign in |
| `GET` | `/api/v1/products` | Retrieve products |
| `GET` | `/api/v1/customers` | Retrieve customers |
| `GET` | `/api/v1/sales-orders` | Retrieve sales orders |
| `POST` | `/api/v1/sales-orders` | Create sales order draft |
| `GET` | `/api/v1/predictions` | Retrieve DSS results |

These examples are not a complete API contract. Refer to the Go backend route registrations and documentation for supported operations.

## Run Tests & Quality Checks

```powershell
# Verify the frontend can build
npm run build

# Inspect scripts available in your local project
npm run
```

If your `package.json` defines testing or linting scripts, run the applicable commands (for example, `npm run test` or `npm run lint`). **Do not assume those scripts exist** before checking.

## Docker Development

If your combined project uses the Docker Compose file in `D:\vanmin\docker-compose.yml`:

```powershell
cd D:\vanmin

docker compose up -d --build

docker compose ps

docker compose logs -f frontend
```

Your actual frontend port and address depend on the Compose port mapping. Localhost-only port bindings are intentionally accessible only from the host computer; testing from a phone requires a deliberate and appropriately secured LAN or HTTPS configuration.

## Deployment

1. Confirm production API origins and routing configurations.
2. Build the application with `npm run build`.
3. Serve the generated `dist/` assets with an appropriate static web server or hosting provider.
4. Configure **SPA fallback** so refreshes of Vue Router URLs return `index.html` rather than a 404.
5. Use **HTTPS**, production authentication, and role-based authorization on the Go API.
6. Verify financial reports, invoice printing, mobile navigation, and integrations in a staging environment before production rollout.

When the frontend is hosted separately from the backend, configure CORS precisely for authorized origins; avoid unrestricted CORS in production.

## Security Notes

- Keep the internal company repository **private** and grant access only to authorized collaborators.
- Do not commit `.env.local`, credentials, secrets, customer data exports, or real company documents.
- Never trust a hidden menu or client-side role check as a security boundary.
- Use the backend for validating user permissions, uploaded SP files, payment status, and invoice ownership.
- Use secure, unpredictable QR access tokens rather than putting sensitive invoice details inside a public QR URL.
- Verify Midtrans payment notifications on the backend; **never** accept client-side success messages as proof of payment.
- Maintain audit logs and enforce rate limits on externally accessible features.

## Roadmap

- [ ] Connect remaining screens to live Go API and PostgreSQL data
- [ ] Complete consistent UI/UX and mobile-responsive workflows
- [ ] Complete financial reporting with print, PDF, and Excel exports
- [ ] Integrate K-Means and Decision Tree DSS results throughout distribution
- [ ] Add invoice QR portal with **Pay Invoice** and **Upload Surat Pesanan (SP)** options
- [ ] Send SP submissions to **Reports** with original-file preview and downloads
- [ ] Test payment processing through **Midtrans Sandbox** using a real phone
- [ ] Perform security testing and production hardening before opening external access

## Resources

- [Vue.js Documentation](https://vuejs.org/guide/introduction.html)
- [Vue Router Documentation](https://router.vuejs.org/)
- [Vite Documentation](https://vite.dev/guide/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [Node.js Documentation](https://nodejs.org/en/docs)
- [Docker Documentation](https://docs.docker.com/)

## Repository & Usage

**Organization:** PT. Furi Anugerah Sejahtera (PT. FAS)  
**Project:** Internal Company Management System  
**Frontend:** Vue.js + Vite + Tailwind CSS  
**Backend:** Go + PostgreSQL  
**Repository:** [Vansyahlee/vanmin](https://github.com/Vansyahlee/vanmin)

This project is intended for **internal company development and operations**. No open-source license is implied. Redistribution, use of company data, and external access require authorization from the repository owner.
