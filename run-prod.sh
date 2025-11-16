#!/bin/bash
echo "🚀 Starting Task Manager API in Production Mode"
echo "==============================================="

if [ ! -f .env ]; then
    echo "❌ .env file not found. Please create it first."
    exit 1
fi

set -a
source .env
set +a

docker-compose -f docker-compose.scalable.yml up -d --build

echo "✅ Production environment is running!"
echo "🌐 Application: http://localhost"
echo "🌐 HTTPS: https://localhost"
echo "📊 Grafana: http://localhost:3001 (admin/admin)"
echo "📈 Prometheus: http://localhost:9090"
echo "📋 Health Check: http://localhost/health"
echo ""
echo "🔍 Port Configuration:"
echo "• PostgreSQL: ${POSTGRES_PORT:-5432}"
echo "• Redis: ${REDIS_PORT:-6379}"
echo "• Backend: ${BACKEND_PORT:-8080}"
echo "• Frontend: ${FRONTEND_PORT:-3000}"
