#!/bin/bash

# Task Manager API - Example curl commands
# Use port 80 (nginx) for all API requests

BASE_URL="http://localhost"
API_URL="${BASE_URL}/api/v1"

echo "🔧 Task Manager API Examples"
echo "============================"
echo ""

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "1️⃣  Register User"
echo -e "${YELLOW}curl -X POST ${API_URL}/auth/register \\${NC}"
curl -X POST "${API_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test.user@example.com",
    "password": "Test@123456",
    "first_name": "Test",
    "last_name": "User"
  }'
echo -e "\n"

echo "2️⃣  Login"
echo -e "${YELLOW}curl -X POST ${API_URL}/auth/login \\${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe.2@example.com",
    "password": "SecurePass@123"
  }')

echo "$LOGIN_RESPONSE" | jq .
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
echo -e "\n${GREEN}✅ Token saved${NC}\n"

echo "3️⃣  Get Profile"
echo -e "${YELLOW}curl -X GET ${API_URL}/users/profile \\${NC}"
curl -X GET "${API_URL}/users/profile" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo -e "\n"

echo "4️⃣  Create Task"
echo -e "${YELLOW}curl -X POST ${API_URL}/tasks \\${NC}"
curl -X POST "${API_URL}/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "title": "Test Task",
    "description": "Testing API via nginx",
    "status": "pending",
    "priority": "medium"
  }' | jq .
echo -e "\n"

echo "5️⃣  Get All Tasks"
echo -e "${YELLOW}curl -X GET ${API_URL}/tasks \\${NC}"
curl -X GET "${API_URL}/tasks" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo -e "\n"

echo "6️⃣  Health Check"
echo -e "${YELLOW}curl -X GET ${BASE_URL}/health \\${NC}"
curl -X GET "${BASE_URL}/health" | jq .
echo -e "\n"

echo "7️⃣  Nginx Health"
echo -e "${YELLOW}curl -X GET ${BASE_URL}/nginx-health \\${NC}"
curl -X GET "${BASE_URL}/nginx-health"
echo -e "\n"

echo "============================"
echo -e "${GREEN}✅ All examples completed!${NC}"
echo ""
echo "💡 Important:"
echo "  - Use port 80 (or omit port)"
echo "  - Backend port 8081 is no longer exposed"
echo "  - All traffic goes through nginx"
echo ""
echo "📚 Documentation:"
echo "  - Scaling: docs/SCALING_GUIDE.md"
echo "  - Admin Cache: docs/ADMIN_CACHE_SECURITY.md"
