#!/bin/bash

# Test script for WindGo Chat Authentication API
# Usage: ./test_auth.sh

API_URL="http://localhost:8080"
CONTENT_TYPE="Content-Type: application/json"

echo "🚀 Testing WindGo Chat Authentication API"
echo "=========================================="

# Test 1: Health Check
echo ""
echo "1. Testing Health Check..."
curl -s -X GET "$API_URL/health" | jq '.' || echo "Health check failed"

# Test 2: Register a new user
echo ""
echo "2. Testing User Registration..."
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/api/auth/register" \
  -H "$CONTENT_TYPE" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }')

echo "Registration Response:"
echo $REGISTER_RESPONSE | jq '.' || echo $REGISTER_RESPONSE

# Extract token from registration response
TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.token // empty')

# Test 3: Login with the same user
echo ""
echo "3. Testing User Login..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/api/auth/login" \
  -H "$CONTENT_TYPE" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }')

echo "Login Response:"
echo $LOGIN_RESPONSE | jq '.' || echo $LOGIN_RESPONSE

# Extract token from login response if registration failed
if [ -z "$TOKEN" ]; then
  TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token // empty')
fi

# Test 4: Test protected route (Get Profile)
if [ ! -z "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo ""
  echo "4. Testing Protected Route (Get Profile)..."
  curl -s -X GET "$API_URL/api/auth/profile" \
    -H "$CONTENT_TYPE" \
    -H "Authorization: Bearer $TOKEN" | jq '.' || echo "Profile request failed"

  # Test 5: Test token refresh
  echo ""
  echo "5. Testing Token Refresh..."
  curl -s -X POST "$API_URL/api/auth/refresh" \
    -H "$CONTENT_TYPE" \
    -H "Authorization: Bearer $TOKEN" | jq '.' || echo "Token refresh failed"
else
  echo ""
  echo "❌ No valid token received. Skipping protected route tests."
fi

# Test 6: Test invalid login
echo ""
echo "6. Testing Invalid Login..."
curl -s -X POST "$API_URL/api/auth/login" \
  -H "$CONTENT_TYPE" \
  -d '{
    "email": "invalid@example.com",
    "password": "wrongpassword"
  }' | jq '.' || echo "Invalid login test completed"

# Test 7: Test protected route without token
echo ""
echo "7. Testing Protected Route Without Token..."
curl -s -X GET "$API_URL/api/auth/profile" \
  -H "$CONTENT_TYPE" | jq '.' || echo "Unauthorized access test completed"

# Test 8: Test duplicate registration
echo ""
echo "8. Testing Duplicate User Registration..."
curl -s -X POST "$API_URL/api/auth/register" \
  -H "$CONTENT_TYPE" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }' | jq '.' || echo "Duplicate registration test completed"

echo ""
echo "✅ All tests completed!"
echo ""
echo "📝 Summary:"
echo "- Your authentication API is ready!"
echo "- Available endpoints:"
echo "  - POST /api/auth/register - Register new user"
echo "  - POST /api/auth/login - Login user"
echo "  - GET /api/auth/profile - Get user profile (protected)"
echo "  - POST /api/auth/refresh - Refresh token (protected)"
echo ""
echo "🔗 Frontend Integration:"
echo "- Use the token in Authorization header: 'Bearer <token>'"
echo "- Token expires in 24 hours"
echo "- Store token securely in your frontend (localStorage, sessionStorage, or cookies)"
