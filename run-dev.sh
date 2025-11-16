#!/bin/bash
echo "🧪 Starting Task Manager API in Development Mode"
echo "================================================"

if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

docker-compose -f docker-compose.scalable.yml up --build postgres redis backend frontend nginx

echo "✅ Development environment is running!"
echo "🌐 Frontend: http://localhost:${FRONTEND_PORT:-3000}"
echo "🔌 Backend API (via nginx): http://localhost:80/api/v1"
echo "📊 Health Check: http://localhost:80/health"
echo "📈 Metrics: http://localhost:80/metrics"
echo "🔧 Nginx Health: http://localhost:80/nginx-health"
