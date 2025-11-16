#!/bin/bash
echo "🛑 Stopping Task Manager API"
echo "============================"

if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

docker-compose -f docker-compose.scalable.yml down

echo "✅ All services stopped"
