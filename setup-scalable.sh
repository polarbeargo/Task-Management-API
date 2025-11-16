#!/bin/bash

echo "🚀 Setting up Scalable Task Management API"
echo "=========================================="

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_status "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    echo "Visit: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    echo "Visit: https://docs.docker.com/compose/install/"
    exit 1
fi

if ! docker info &> /dev/null; then
    print_error "Docker daemon is not running. Please start Docker first."
    exit 1
fi

print_success "Docker and Docker Compose are available"

print_status "Checking for port conflicts..."

check_port() {
    local port=$1
    local service=$2
    if lsof -i :$port > /dev/null 2>&1; then
        print_warning "Port $port is already in use (needed for $service)"
        return 1
    fi
    return 0
}

POSTGRES_PORT=5432
REDIS_PORT=6379
BACKEND_PORT=8080
FRONTEND_PORT=3000

if ! check_port $POSTGRES_PORT "PostgreSQL"; then
    POSTGRES_PORT=5433
    print_status "Using alternative PostgreSQL port: $POSTGRES_PORT"
fi

if ! check_port $REDIS_PORT "Redis"; then
    REDIS_PORT=6380
    print_status "Using alternative Redis port: $REDIS_PORT"
fi

if ! check_port $BACKEND_PORT "Backend API"; then
    BACKEND_PORT=8081
    print_status "Using alternative Backend port: $BACKEND_PORT"
fi

if ! check_port $FRONTEND_PORT "Frontend"; then
    FRONTEND_PORT=3001
    print_status "Using alternative Frontend port: $FRONTEND_PORT"
fi

print_success "Port conflict check completed"

if command -v git &> /dev/null; then
    print_success "Git is available"
else
    print_warning "Git is not installed. You may need it for version control."
fi

print_status "Setting up environment configuration..."

if [ ! -f .env ]; then
    print_status "Creating .env file with default values..."
    cat > .env << EOF
# Database Configuration
DB_HOST=postgres
DB_PORT=$POSTGRES_PORT
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=task_manager
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=1h
DB_CONN_MAX_IDLE_TIME=30m

# Redis Configuration
REDIS_HOST=redis
REDIS_PORT=$REDIS_PORT
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5
REDIS_MAX_RETRIES=3

# Server Configuration
HOST=0.0.0.0
PORT=$BACKEND_PORT
ENVIRONMENT=production
SCALABLE_MODE=true
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
IDLE_TIMEOUT=60s

# JWT Configuration (CHANGE IN PRODUCTION!)
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY=24h
JWT_REFRESH_EXPIRY=720h

# Worker Configuration
WORKER_CONCURRENCY=4
WORKER_POLL_INTERVAL=5s

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPM=100
RATE_LIMIT_BURST=10

# CORS Configuration
CORS_ORIGINS=http://localhost:$FRONTEND_PORT,http://localhost:$BACKEND_PORT
CORS_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_HEADERS=Origin,Content-Type,Authorization

# Monitoring
METRICS_ENABLED=true
HEALTH_CHECK_ENABLED=true

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Port Configuration (for docker-compose)
POSTGRES_PORT=$POSTGRES_PORT
REDIS_PORT=$REDIS_PORT
BACKEND_PORT=$BACKEND_PORT
FRONTEND_PORT=$FRONTEND_PORT
EOF
    print_success "Created .env file with port configuration: PostgreSQL:$POSTGRES_PORT, Redis:$REDIS_PORT, Backend:$BACKEND_PORT, Frontend:$FRONTEND_PORT"
    print_warning "Please review and update the .env file for production use!"
else
    print_success ".env file already exists"
    if ! grep -q "POSTGRES_PORT=" .env; then
        echo "POSTGRES_PORT=$POSTGRES_PORT" >> .env
        echo "REDIS_PORT=$REDIS_PORT" >> .env  
        echo "BACKEND_PORT=$BACKEND_PORT" >> .env
        echo "FRONTEND_PORT=$FRONTEND_PORT" >> .env
        print_status "Updated .env with port configuration"
    fi
fi

print_status "Creating necessary directories..."

mkdir -p logs
mkdir -p test-results
mkdir -p monitoring/grafana/dashboards
mkdir -p monitoring/prometheus

print_success "Directories created"

print_status "Pulling Docker images..."

export POSTGRES_PORT
export REDIS_PORT  
export BACKEND_PORT
export FRONTEND_PORT

docker-compose -f docker-compose.scalable.yml pull

print_success "Docker images pulled successfully"

print_status "Building application images..."

docker-compose -f docker-compose.scalable.yml build --no-cache

if [ $? -eq 0 ]; then
    print_success "Application images built successfully"
else
    print_error "Failed to build application images"
    exit 1
fi

print_status "Setting up database..."

print_status "Starting PostgreSQL container on port $POSTGRES_PORT..."
if ! docker-compose -f docker-compose.scalable.yml up -d postgres; then
    print_error "Failed to start PostgreSQL container"
    print_error "This might be due to port conflicts. Please check:"
    echo "  1. Stop any local PostgreSQL services: brew services stop postgresql"
    echo "  2. Kill any processes using port $POSTGRES_PORT: sudo lsof -ti:$POSTGRES_PORT | xargs kill -9"
    echo "  3. Try running the setup again"
    exit 1
fi

print_status "Waiting for PostgreSQL to be ready..."
sleep 10

max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose -f docker-compose.scalable.yml exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        break
    fi
    attempt=$((attempt + 1))
    print_status "Waiting for PostgreSQL... (attempt $attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    print_error "PostgreSQL failed to start within expected time"
    exit 1
fi

print_success "PostgreSQL is ready"

docker-compose -f docker-compose.scalable.yml stop postgres

print_status "Setting executable permissions on scripts..."

chmod +x run-dev.sh
chmod +x run-prod.sh
chmod +x scale.sh
chmod +x stop.sh

print_success "Script permissions set"

print_success "🎉 Scalable Task Management API setup completed!"
echo ""
echo "📋 Next Steps:"
echo "==============="
echo ""
echo "1. 📝 Review and update the .env file:"
echo "   ${YELLOW}nano .env${NC}"
echo ""
echo "2. 🧪 Start in development mode:"
echo "   ${YELLOW}./run-dev.sh${NC}"
echo ""
echo "3. 🚀 Or start in production mode:"
echo "   ${YELLOW}./run-prod.sh${NC}"
echo ""
echo "4. ⚡ Scale the backend:"
echo "   ${YELLOW}./scale.sh 3${NC}  (scales to 3 instances)"
echo ""
echo "5. 🛑 Stop all services:"
echo "   ${YELLOW}./stop.sh${NC}"
echo ""
echo "📊 Monitoring URLs (after startup):"
echo "===================================="
echo "• Application: ${BLUE}http://localhost${NC} (production) or ${BLUE}http://localhost:$FRONTEND_PORT${NC} (dev)"
echo "• API Health: ${BLUE}http://localhost:$BACKEND_PORT/health${NC} (dev) or ${BLUE}http://localhost/health${NC} (prod)"
echo "• Grafana: ${BLUE}http://localhost:3001${NC} (admin/admin)"
echo "• Prometheus: ${BLUE}http://localhost:9090${NC}"
echo ""
if [ "$POSTGRES_PORT" != "5432" ] || [ "$REDIS_PORT" != "6379" ] || [ "$BACKEND_PORT" != "8080" ] || [ "$FRONTEND_PORT" != "3000" ]; then
    print_warning "Non-standard ports detected due to conflicts:"
    echo "• PostgreSQL: $POSTGRES_PORT (default: 5432)"
    echo "• Redis: $REDIS_PORT (default: 6379)" 
    echo "• Backend: $BACKEND_PORT (default: 8080)"
    echo "• Frontend: $FRONTEND_PORT (default: 3000)"
    echo ""
fi
echo ""
print_warning "Remember to change JWT_SECRET and database passwords for production!"