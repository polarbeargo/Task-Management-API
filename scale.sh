#!/bin/bash

# Scale backend instances for the Task Manager API
# Usage: ./scale.sh [number_of_replicas]
# Example: ./scale.sh 3

REPLICAS=${1:-2}

echo "⚡ Scaling backend to $REPLICAS replicas"
echo "========================================"

if [ -f .env ]; then
    set -a
    source .env
    set +a
    echo "✅ Loaded environment variables from .env"
fi

if [ ! -f docker-compose.scalable.yml ]; then
    echo "❌ Error: docker-compose.scalable.yml not found"
    exit 1
fi

echo "📦 Scaling backend service..."
docker-compose -f docker-compose.scalable.yml up -d --scale backend=$REPLICAS --no-recreate

sleep 2

echo ""
echo "✅ Backend scaled to $REPLICAS instances"
echo ""
echo "📊 Current container status:"
docker-compose -f docker-compose.scalable.yml ps backend

echo ""
echo "🔍 Backend instances:"
docker ps --filter "name=task-management-api-backend" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "💡 Tips:"
echo "  - Access via nginx: http://localhost:80"
echo "  - View logs: docker-compose -f docker-compose.scalable.yml logs -f backend"
echo "  - Stop all: docker-compose -f docker-compose.scalable.yml down"
echo "  - Scale to N: ./scale.sh N"
