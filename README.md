# 🌸 Flower Supply Catalog

A fullstack Go web application for managing and displaying a flower supply product catalog.

## Tech Stack

- **Backend:** Go 1.22+ with Fiber framework
- **Frontend:** htmx + Tailwind CSS
- **Database:** PostgreSQL 15+
- **Image Storage:** Cloudinary
- **Deployment:** Docker + Railway

## Project Structure

```
.
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP handlers (public, admin, auth, upload)
│   ├── services/        # Business logic layer
│   ├── repositories/    # Data access layer
│   ├── models/          # Data models
│   └── middleware/      # HTTP middleware
├── web/
│   ├── templates/       # HTML templates
│   └── static/          # Static assets (CSS, JS, images)
├── migrations/          # Database migrations
├── docker/              # Dockerfiles
├── docker-compose.yml   # Development environment
├── go.mod               # Go dependencies
├── Makefile             # Common commands
└── .env.example         # Environment variables template
```

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Docker and Docker Compose
- Cloudinary account (for image storage)

### Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd aslam-flower/app
   ```

2. **Copy environment variables**
   ```bash
   cp .env.example .env
   ```

3. **Edit `.env` file** with your configuration:
   - Cloudinary credentials
   - Database URL (for local development)
   - JWT secret (generate a random 32-character string)
   - WhatsApp number

4. **Start development environment**
   ```bash
   make docker-up
   ```

   Or manually:
   ```bash
   docker-compose up -d
   ```

5. **Run database migrations**
   ```bash
   make migrate-up
   ```

6. **Access the application**
   - Public catalog: http://localhost:3000
   - Admin panel: http://localhost:3000/admin

### Development

**Run locally (without Docker):**
```bash
make run
```

**Run with hot reload:**
```bash
make dev
```

**Run tests:**
```bash
make test
```

**View logs:**
```bash
make docker-logs
```

### Building for Production

**Build Docker image:**
```bash
make docker-build
```

**Build Go binary:**
```bash
make build
```

## Available Commands

See all available commands:
```bash
make help
```

Common commands:
- `make build` - Build the application
- `make run` - Run the application
- `make dev` - Run with hot reload
- `make test` - Run tests
- `make docker-up` - Start Docker services
- `make docker-down` - Stop Docker services
- `make docker-logs` - View logs
- `make deps` - Download dependencies
- `make fmt` - Format code
- `make vet` - Run go vet

## Environment Variables

See `.env.example` for all required environment variables.

Required:
- `DATABASE_URL` - PostgreSQL connection string
- `CLOUDINARY_CLOUD_NAME` - Cloudinary cloud name
- `CLOUDINARY_API_KEY` - Cloudinary API key
- `CLOUDINARY_API_SECRET` - Cloudinary API secret
- `JWT_SECRET` - Secret key for JWT tokens (min 32 characters)

Optional:
- `PORT` - Server port (default: 3000)
- `ENV` - Environment (development/production)
- `WHATSAPP_NUMBER` - Seller's WhatsApp number
- `ADMIN_USERNAME` - Default admin username (for seeding)
- `ADMIN_PASSWORD` - Default admin password (for seeding)

## Architecture

The application follows a layered architecture:

1. **Handlers** - HTTP request/response handling
2. **Services** - Business logic and orchestration
3. **Repositories** - Data access layer
4. **Models** - Data structures

See `AGENTS.md` for detailed architecture documentation.

## License

Private project - All rights reserved

