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
RED='\033[0;31m'
NC='\033[0m'

# Generate unique email for testing
TIMESTAMP=$(date +%s)
TEST_EMAIL="test.user.${TIMESTAMP}@example.com"
TEST_USERNAME="test_user_${TIMESTAMP}"

echo "1️⃣  Register User"
echo -e "${YELLOW}curl -X POST ${API_URL}/auth/register \\${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "${API_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"${TEST_USERNAME}\",
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"Test@123456\",
    \"first_name\": \"Test\",
    \"last_name\": \"User\"
  }")

echo "$REGISTER_RESPONSE" | jq .

# Check if registration was successful
if echo "$REGISTER_RESPONSE" | jq -e '.user.id' > /dev/null 2>&1; then
  echo -e "${GREEN}✅ Registration successful${NC}\n"
else
  echo -e "${RED}❌ Registration failed${NC}\n"
fi

echo "2️⃣  Login with New User"
echo -e "${YELLOW}curl -X POST ${API_URL}/auth/login \\${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"Test@123456\"
  }")

echo "$LOGIN_RESPONSE" | jq .
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')

# Check if login was successful
if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
  echo -e "\n${GREEN}✅ Login successful - Token saved${NC}\n"
else
  echo -e "\n${RED}❌ Login failed - No token received${NC}"
  echo -e "${YELLOW}💡 Trying with existing user (jane@example.com)${NC}\n"
  
  # Fallback to existing user
  LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/auth/login" \
    -H "Content-Type: application/json" \
    -d '{
      "email": "jane@example.com",
      "password": "SecurePass@456"
    }')
  
  echo "$LOGIN_RESPONSE" | jq .
  TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
  
  if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
    echo -e "\n${GREEN}✅ Login successful with existing user${NC}\n"
  else
    echo -e "\n${RED}❌ Login failed - Cannot continue with authenticated requests${NC}\n"
    TOKEN=""
  fi
fi

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "3️⃣  Get Profile"
  echo -e "${YELLOW}curl -X GET ${API_URL}/users/profile \\${NC}"
  PROFILE_RESPONSE=$(curl -s -X GET "${API_URL}/users/profile" \
    -H "Authorization: Bearer $TOKEN")
  echo "$PROFILE_RESPONSE" | jq .
  
  if echo "$PROFILE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Profile retrieved successfully${NC}\n"
  else
    echo -e "${RED}❌ Failed to get profile${NC}\n"
  fi

  echo "4️⃣  Create Task"
  echo -e "${YELLOW}curl -X POST ${API_URL}/tasks \\${NC}"
  TASK_RESPONSE=$(curl -s -X POST "${API_URL}/tasks" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
      "title": "Test Task",
      "description": "Testing API via nginx",
      "status": "pending",
      "priority": "medium"
    }')
  echo "$TASK_RESPONSE" | jq .
  
  if echo "$TASK_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Task created successfully${NC}\n"
    TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.id')
  else
    echo -e "${RED}❌ Failed to create task${NC}\n"
  fi

  echo "5️⃣  Get All Tasks"
  echo -e "${YELLOW}curl -X GET ${API_URL}/tasks \\${NC}"
  TASKS_RESPONSE=$(curl -s -X GET "${API_URL}/tasks" \
    -H "Authorization: Bearer $TOKEN")
  echo "$TASKS_RESPONSE" | jq .
  
  if echo "$TASKS_RESPONSE" | jq -e 'length' > /dev/null 2>&1; then
    TASK_COUNT=$(echo "$TASKS_RESPONSE" | jq 'length')
    echo -e "${GREEN}✅ Retrieved ${TASK_COUNT} tasks${NC}\n"
  else
    echo -e "${RED}❌ Failed to retrieve tasks${NC}\n"
  fi

  # Bonus: Update task if we created one
  if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "null" ]; then
    echo "6️⃣  Update Task"
    echo -e "${YELLOW}curl -X PUT ${API_URL}/tasks/${TASK_ID} \\${NC}"
    UPDATE_RESPONSE=$(curl -s -X PUT "${API_URL}/tasks/${TASK_ID}" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{
        "title": "Updated Test Task",
        "status": "in_progress"
      }')
    echo "$UPDATE_RESPONSE" | jq .
    
    if echo "$UPDATE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
      echo -e "${GREEN}✅ Task updated successfully${NC}\n"
    else
      echo -e "${RED}❌ Failed to update task${NC}\n"
    fi
  fi
else
  echo -e "${RED}⚠️  Skipping authenticated requests (no valid token)${NC}\n"
fi

echo "7️⃣  Health Check"
echo -e "${YELLOW}curl -X GET ${BASE_URL}/health \\${NC}"
HEALTH_RESPONSE=$(curl -s -X GET "${BASE_URL}/health")
echo "$HEALTH_RESPONSE" | jq .

if echo "$HEALTH_RESPONSE" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
  echo -e "${GREEN}✅ System is healthy${NC}\n"
else
  echo -e "${RED}❌ System health check failed${NC}\n"
fi

echo "8️⃣  Nginx Health"
echo -e "${YELLOW}curl -X GET ${BASE_URL}/nginx-health \\${NC}"
NGINX_RESPONSE=$(curl -s -X GET "${BASE_URL}/nginx-health")
echo "$NGINX_RESPONSE"
echo -e "\n"

echo "============================"
echo -e "${GREEN}✅ All API tests completed!${NC}"
echo ""
echo "📊 Test Summary:"
echo "  - Registration: ${TEST_EMAIL}"
if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "  - Authentication: ✅ Success"
  echo "  - Token received: ${TOKEN:0:20}..."
else
  echo "  - Authentication: ❌ Failed"
fi
if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "null" ]; then
  echo "  - Task created: ${TASK_ID}"
fi
echo ""
echo "💡 Important:"
echo "  - Use port 80 (or omit port)"
echo "  - Backend port 8081 is no longer exposed"
echo "  - All traffic goes through nginx"
echo ""
echo "📚 Documentation:"
echo "  - Scaling: docs/SCALING_GUIDE.md"
echo "  - Admin Cache: docs/ADMIN_CACHE_SECURITY.md"
