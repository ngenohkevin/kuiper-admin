# Ganymede Admin Dashboard

A modern, responsive admin dashboard for managing a Supabase database. Built with Golang, HTMX, Alpine.js, Tailwind CSS, and templ.


## Features

- **Server-side Rendering**: Lightweight and fast with Go and templ templates
- **Modern UI**: Clean and responsive interface with Tailwind CSS
- **Interactive UI**: Dynamic interactions without JavaScript thanks to HTMX
- **Simple State Management**: Alpine.js for client-side state management
- **Database Integration**: Direct connection to Supabase PostgreSQL database
- **CRUD Operations**: Create, read, update, and delete for all entities
- **Relationship Management**: Handle relationships between entities

## Technologies Used

- **Backend**:
  - [Go](https://golang.org/) - Server-side language
  - [Chi Router](https://github.com/go-chi/chi) - HTTP routing
  - [templ](https://github.com/a-h/templ) - HTML templating language for Go
  - [pgx](https://github.com/jackc/pgx) - PostgreSQL driver

- **Frontend**:
  - [HTMX](https://htmx.org/) - Dynamic HTML without JavaScript
  - [Alpine.js](https://alpinejs.dev/) - Minimal JavaScript framework
  - [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework

- **Database**:
  - [Supabase](https://supabase.com/) - PostgreSQL database service

## Project Structure

```
kuiper_admin/
├── cmd/                  # Application entry point
├── internal/             # Internal packages
│   ├── database/         # Database connection and utilities
│   ├── handlers/         # HTTP request handlers
│   ├── models/           # Data models
│   └── templates/        # templ HTML templates
├── migrations/           # Database migrations
├── web/                  # Web assets
│   └── static/           # Static files (CSS, JS)
└── Makefile              # Build automation
```

## Getting Started

### Prerequisites

- Go 1.21+
- templ CLI
- PostgreSQL database (or Supabase account)

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/kuiper_admin.git
   cd kuiper_admin
   ```

2. Set up the development environment:
   ```bash
   make setup_dev
   ```

3. Build and run the application:
   ```bash
   make serve
   ```

4. Access the dashboard at [http://localhost:8090](http://localhost:8090)

## Development

- **Generate templ files**: `make templ`
- **Build the application**: `make build`
- **Run the application**: `make run`
- **Clean build files**: `make clean`
- **Install dependencies**: `make deps`
- **Run tests**: `make test`
- **Display help**: `make help`

## Configuration

The application is configured using environment variables in the `.env` file:

```env
DB_NAME=ganymede
DB_USER=postgres.username
DB_PASSWORD=yourpassword
DB_HOST=yourhost
DB_PORT=5432
DATABASE_URL=postgresql://postgres.username:yourpassword@yourhost:5432/postgres
PORT=8090
```

## Entities

The dashboard manages the following entities:

- **Categories**: Product categories with hierarchical relationships
- **Products**: Items for sale with associated categories
- **Reviews**: Customer reviews for products

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [HTMX](https://htmx.org/) for simplifying client-server interactions
- [Alpine.js](https://alpinejs.dev/) for lightweight JavaScript
- [Tailwind CSS](https://tailwindcss.com/) for utility-first CSS
- [templ](https://github.com/a-h/templ) for the innovative templating language
- [Supabase](https://supabase.com/) for the robust PostgreSQL platform
# kuiper-admin
