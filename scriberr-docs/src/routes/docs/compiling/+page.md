---
---

# Compiling from Source

This guide covers building Scriberr from source code for development, customization, or deployment on unsupported platforms.

## Prerequisites

### System Requirements

- **Go**: Version 1.21 or later
- **Git**: For cloning the repository
- **Make**: For build automation (optional)
- **Docker**: For containerized builds (optional)

### Development Tools

- **Node.js**: Version 18 or later (for frontend development)
- **npm** or **yarn**: Package manager for frontend dependencies
- **Python**: Version 3.8+ (for WhisperX dependencies)

## Getting the Source Code

### Clone the Repository

```bash
# Clone the main repository
git clone https://github.com/noeticgeek/scriberr.git
cd scriberr

# Checkout the latest release
git checkout v1.0.0-beta1
```

### Repository Structure

```
scriberr/
├── cmd/                    # Main application entry points
├── internal/              # Internal packages
│   ├── database/          # Database operations
│   ├── handlers/          # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   ├── models/            # Data models
│   ├── summary_tasks/     # Summarization tasks
│   └── tasks/             # Background tasks
├── scriberr-frontend/     # SvelteKit frontend
├── scriberr-docs/         # Documentation site
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
└── README.md              # Project documentation
```

## Building the Backend

### Basic Build

```bash
# Build the main binary
go build -o scriberr ./cmd/scriberr

# Build with specific Go version
go build -ldflags="-s -w" -o scriberr ./cmd/scriberr
```

### Cross-Platform Builds

```bash
# Build for Linux AMD64
GOOS=linux GOARCH=amd64 go build -o scriberr-linux-amd64 ./cmd/scriberr

# Build for Linux ARM64
GOOS=linux GOARCH=arm64 go build -o scriberr-linux-arm64 ./cmd/scriberr

# Build for macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o scriberr-darwin-amd64 ./cmd/scriberr

# Build for macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o scriberr-darwin-arm64 ./cmd/scriberr

# Build for Windows AMD64
GOOS=windows GOARCH=amd64 go build -o scriberr-windows-amd64.exe ./cmd/scriberr
```

### Optimized Builds

```bash
# Build with optimizations and stripped symbols
go build -ldflags="-s -w" -o scriberr ./cmd/scriberr

# Build with specific tags
go build -tags="cuda" -o scriberr ./cmd/scriberr

# Build with race detection (for debugging)
go build -race -o scriberr ./cmd/scriberr
```

## Building the Frontend

### Install Dependencies

```bash
# Navigate to frontend directory
cd scriberr-frontend

# Install dependencies
npm install

# Or using yarn
yarn install
```

### Development Build

```bash
# Start development server
npm run dev

# Build for development
npm run build

# Preview production build
npm run preview
```

### Production Build

```bash
# Build for production
npm run build

# The built files will be in the dist/ directory
```

## Building with Docker

### Multi-stage Docker Build

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Install build dependencies
RUN apk add --no-cache git make

# Build the binary
RUN go build -ldflags="-s -w" -o scriberr ./cmd/scriberr

# Frontend build stage
FROM node:18-alpine AS frontend-builder

WORKDIR /app
COPY scriberr-frontend/package*.json ./
RUN npm ci --only=production

COPY scriberr-frontend/ ./
RUN npm run build

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/scriberr .

# Copy frontend assets
COPY --from=frontend-builder /app/dist ./static

EXPOSE 8080

CMD ["./scriberr"]
```

### Build Docker Image

```bash
# Build the image
docker build -t scriberr:latest .

# Run the container
docker run -p 8080:8080 scriberr:latest
```

## Development Setup

### Local Development Environment

1. **Backend Development**:
   ```bash
   # Install Go dependencies
   go mod download

   # Run tests
   go test ./...

   # Run with hot reload (using air)
   air
   ```

2. **Frontend Development**:
   ```bash
   cd scriberr-frontend
   npm run dev
   ```

3. **Database Setup**:
   ```bash
   # Initialize SQLite database
   go run cmd/scriberr/main.go --init-db
   ```

### Environment Configuration

Create a `.env` file for development:

```env
# Development settings
SCRIBERR_ENV=development
SCRIBERR_PORT=8080
SCRIBERR_HOST=localhost

# Database
SCRIBERR_DB_PATH=./data/scriberr.db

# Models
SCRIBERR_MODEL_SIZE=base
SCRIBERR_DEVICE=cpu

# AI Integration
OPENAI_API_KEY=your_openai_key_here
OLLAMA_URL=http://localhost:11434
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test ./internal/handlers -v
```

### Frontend Tests

```bash
cd scriberr-frontend

# Run unit tests
npm test

# Run tests with coverage
npm run test:coverage

# Run end-to-end tests
npm run test:e2e
```

## Customization

### Adding New Features

1. **Backend Extensions**:
   - Add new handlers in `internal/handlers/`
   - Create new models in `internal/models/`
   - Extend database operations in `internal/database/`

2. **Frontend Extensions**:
   - Add new components in `src/lib/components/`
   - Create new pages in `src/routes/`
   - Extend stores in `src/lib/stores.ts`

### Configuration Options

Modify `internal/config/config.go` to add new configuration options:

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Models   ModelsConfig   `yaml:"models"`
    Storage  StorageConfig  `yaml:"storage"`
    AI       AIConfig       `yaml:"ai"`
    // Add your custom config here
}
```

## Deployment

### Production Build

```bash
# Build optimized binary
go build -ldflags="-s -w" -o scriberr ./cmd/scriberr

# Build frontend
cd scriberr-frontend
npm run build

# Copy frontend assets to binary directory
cp -r dist/* ../static/
```

### Systemd Service

Create `/etc/systemd/system/scriberr.service`:

```ini
[Unit]
Description=Scriberr Audio Transcription Service
After=network.target

[Service]
Type=simple
User=scriberr
WorkingDirectory=/opt/scriberr
ExecStart=/opt/scriberr/scriberr
Restart=always
RestartSec=5
Environment=SCRIBERR_PORT=8080
Environment=SCRIBERR_HOST=0.0.0.0

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl enable scriberr
sudo systemctl start scriberr
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  scriberr:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./audio:/app/audio
    environment:
      - SCRIBERR_PORT=8080
      - SCRIBERR_HOST=0.0.0.0
    restart: unless-stopped
```

## Troubleshooting

### Common Build Issues

**Go Module Issues**:
```bash
# Clean module cache
go clean -modcache

# Update dependencies
go mod tidy
go mod download
```

**Frontend Build Issues**:
```bash
# Clear npm cache
npm cache clean --force

# Remove node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

**Docker Build Issues**:
```bash
# Build without cache
docker build --no-cache -t scriberr:latest .

# Check Docker daemon
docker system prune -a
```

### Performance Optimization

1. **Binary Size**:
   ```bash
   # Strip debug symbols
   go build -ldflags="-s -w" -o scriberr ./cmd/scriberr
   
   # Use UPX compression
   upx --best scriberr
   ```

2. **Build Time**:
   ```bash
   # Use Go build cache
   export GOCACHE=/tmp/go-cache
   
   # Parallel builds
   go build -p 4 ./cmd/scriberr
   ```

## Contributing

### Development Workflow

1. **Fork the Repository**:
   ```bash
   git clone https://github.com/your-username/scriberr.git
   cd scriberr
   git remote add upstream https://github.com/noeticgeek/scriberr.git
   ```

2. **Create Feature Branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes and Test**:
   ```bash
   # Make your changes
   go test ./...
   npm test  # in frontend directory
   ```

4. **Submit Pull Request**:
   ```bash
   git add .
   git commit -m "Add your feature description"
   git push origin feature/your-feature-name
   ```

### Code Style

- Follow Go formatting: `gofmt -s -w .`
- Use `golint` for code quality
- Follow SvelteKit conventions for frontend
- Write tests for new features

## Support

For build issues or questions:

1. Check the [GitHub Issues](https://github.com/noeticgeek/scriberr/issues)
2. Review the [Contributing Guidelines](https://github.com/noeticgeek/scriberr/blob/main/CONTRIBUTING.md)
3. Join the [Discussions](https://github.com/noeticgeek/scriberr/discussions) 