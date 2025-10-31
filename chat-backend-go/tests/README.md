# WindGo Chat - Testing Guide

This directory contains comprehensive tests for the WindGo Chat backend, including integration tests, load tests, and manual QA procedures.

## Prerequisites

- Go 1.24+
- PostgreSQL running with `windgo_chat` and `windgo_chat_test` databases
- Backend server running on `http://localhost:8080`
- Valid JWT tokens for authenticated requests

## Test Types

### 1. Integration Tests

Integration tests verify the core CRUD functionality for rooms with proper admin/non-admin access controls.

**Location:** `handlers/room_handlers_test.go`

**Setup Test Database:**
```bash
# Create test database
psql -U postgres -c "CREATE DATABASE windgo_chat_test;"

# Run migrations (handled automatically by tests)
```

**Run Tests:**
```bash
cd chat-backend-go

# Run all tests
go test ./handlers/... -v

# Run specific test
go test ./handlers/ -run TestCreateRoom_AsAdmin -v

# Run with coverage
go test ./handlers/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Tests Included:**
- `TestCreateRoom_AsAdmin` - Verifies admin can create rooms
- `TestCreateRoom_AsNonAdmin` - Verifies non-admins are denied
- `TestUpdateRoom_AsAdmin` - Verifies room name updates
- `TestDeleteRoom_AsAdmin` - Verifies soft delete functionality
- `TestGetRoomByID` - Verifies room retrieval
- `TestCreateRoom_EmptyName` - Verifies validation

### 2. Soft Delete Verification Tests

Automated tests to verify soft delete behavior for rooms and messages.

**Location:** `tests/verify_soft_delete.go`

**Run Tests:**
```bash
cd chat-backend-go/tests

# Run with default settings
go run verify_soft_delete.go

# Run with custom settings
go run verify_soft_delete.go \
  -url http://localhost:8080 \
  -admin-email admin@windgo.com \
  -admin-password admin123
```

**What It Tests:**
- Creates a room and messages
- Soft deletes the room
- Verifies room no longer appears in API responses
- Verifies messages are preserved with DeletedAt timestamps
- Validates soft delete for messages

**Expected Output:**
```
=== Soft Delete Verification Tests ===
Testing against: http://localhost:8080

1. Logging in as admin...
✅ Login successful

2. Testing room soft delete...
   Creating test room...
✅ Room created with ID: 5
   Adding messages to room...
✅ Messages created with IDs: 10, 11
   Verifying room appears in list...
✅ Room appears in list
   Deleting room (soft delete)...
✅ Room deleted successfully
   Verifying room no longer appears in list...
✅ Room no longer appears in list (soft delete working)

3. Testing message soft delete...
[... similar output ...]

✅ All soft delete tests passed successfully!
```

### 3. WebSocket Load Tests

Performance tests to verify WebSocket handling under load with multiple concurrent clients.

**Location:** `tests/websocket_load.go`

**Requirements:**
```bash
go get github.com/gorilla/websocket
```

**Run Tests:**
```bash
cd chat-backend-go/tests

# Basic load test (10 clients for 60 seconds)
go run websocket_load.go -token YOUR_JWT_TOKEN

# Heavy load test (100 clients for 5 minutes)
go run websocket_load.go \
  -token YOUR_JWT_TOKEN \
  -clients 100 \
  -duration 300s \
  -room 1 \
  -rate 10s

# Verbose output
go run websocket_load.go \
  -token YOUR_JWT_TOKEN \
  -clients 50 \
  -verbose
```

**Parameters:**
- `-clients` - Number of concurrent WebSocket clients (default: 10)
- `-duration` - Test duration (default: 60s)
- `-url` - WebSocket server URL (default: ws://localhost:8080/ws)
- `-token` - JWT token for authentication (required)
- `-room` - Room ID to join (default: 1)
- `-rate` - Message sending rate per client (default: 5s)
- `-verbose` - Enable detailed logging

**Expected Output:**
```
Starting WebSocket load test with 50 clients for 1m0s
Server URL: ws://localhost:8080/ws
Room ID: 1
Message rate: 5s per client

[... test runs ...]

=== Load Test Results ===
Duration: 1m0s
Number of clients: 50

Total Messages Sent: 600
Total Messages Received: 600
Total Errors: 0
Messages/sec (sent): 10.00
Messages/sec (received): 10.00

✅ All operations completed successfully!
```

**Performance Benchmarks:**
- **Target:** 100 concurrent clients with < 1% error rate
- **Expected throughput:** 100+ messages/sec
- **Max latency:** < 500ms for message delivery

### 4. Manual QA Test Plan

Comprehensive manual testing procedures covering all features.

**Location:** `tests/MANUAL_QA_TEST_PLAN.md`

**Test Categories:**
- Authentication (Email/Password, GitHub OAuth)
- Room Management (Create, Read, Update, Delete)
- Message Operations (Send, Edit, Delete, Reply)
- User Search
- WebSocket Real-time Features
- Soft Delete Verification
- Input Validation
- Error Handling
- Performance & Load

**How to Use:**
1. Open `MANUAL_QA_TEST_PLAN.md`
2. Follow test procedures step-by-step
3. Document results in the checklist
4. Report any issues found

**Test Execution:**
```bash
# View the test plan
cat tests/MANUAL_QA_TEST_PLAN.md

# Example: Test authentication
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@windgo.com","password":"admin123"}' \
  | jq

# Example: Test room creation
curl -X POST http://localhost:8080/api/v1/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Room"}' \
  | jq
```

## Getting JWT Tokens for Testing

### Method 1: Via CLI
```bash
# Login and extract token from credentials file
cd cli
go run ./cmd/windgo
# After login, token is stored in ~/.config/windgo/credentials.json
cat ~/.config/windgo/credentials.json | jq -r '.token'
```

### Method 2: Via API
```bash
# Login via API
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@windgo.com","password":"admin123"}' \
  | jq -r '.token'

# Save to environment variable
export TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@windgo.com","password":"admin123"}' \
  | jq -r '.token')
```

## Running All Tests

### Quick Test Suite
```bash
#!/bin/bash
# Run this script to execute all automated tests

echo "=== WindGo Chat Test Suite ==="

# 1. Integration Tests
echo "\n1. Running integration tests..."
cd chat-backend-go
go test ./handlers/... -v

# 2. Soft Delete Tests
echo "\n2. Running soft delete verification..."
cd tests
go run verify_soft_delete.go

# 3. WebSocket Load Tests
echo "\n3. Running WebSocket load tests..."
# Get admin token
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@windgo.com","password":"admin123"}' \
  | jq -r '.token')

go run websocket_load.go \
  -token $TOKEN \
  -clients 20 \
  -duration 30s

echo "\n=== All Automated Tests Complete ==="
echo "Run manual QA tests from MANUAL_QA_TEST_PLAN.md"
```

## Continuous Integration

### GitHub Actions Example
```yaml
name: Run Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: windgo_chat_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install dependencies
        run: |
          cd chat-backend-go
          go mod download

      - name: Run integration tests
        run: |
          cd chat-backend-go
          go test ./handlers/... -v
        env:
          DB_HOST: localhost
          DB_PORT: 5432
          DB_USER: postgres
          DB_PASSWORD: postgres
          DB_NAME: windgo_chat_test
          JWT_SECRET: test_secret
```

## Test Coverage

### Generate Coverage Reports
```bash
cd chat-backend-go

# Generate coverage for all packages
go test ./... -coverprofile=coverage.out -covermode=atomic

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage Goals
- **Handlers:** > 80%
- **Models:** > 70%
- **Utils:** > 85%
- **Middleware:** > 75%

## Troubleshooting

### Test Database Connection Issues
```bash
# Check if PostgreSQL is running
systemctl status postgresql  # Linux
brew services list | grep postgresql  # macOS

# Create test database if missing
psql -U postgres -c "CREATE DATABASE windgo_chat_test;"

# Grant permissions
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE windgo_chat_test TO postgres;"
```

### WebSocket Test Connection Issues
```bash
# Verify server is running
curl http://localhost:8080/health

# Check WebSocket endpoint
curl -i -N -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: test" \
  http://localhost:8080/ws?token=YOUR_TOKEN
```

### Token Expiration
If tests fail with "unauthorized" errors, tokens may have expired (24h default). Generate new tokens:
```bash
export TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@windgo.com","password":"admin123"}' \
  | jq -r '.token')
```

## Best Practices

1. **Run tests before commits:** Always run integration tests before pushing code
2. **Use test database:** Never run tests against production database
3. **Clean up test data:** Tests should create and clean up their own data
4. **Document new tests:** Add documentation when adding new test files
5. **Monitor performance:** Track WebSocket load test results over time
6. **Update manual QA:** Keep MANUAL_QA_TEST_PLAN.md current with new features

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Fiber Testing Guide](https://docs.gofiber.io/guide/testing)
- [GORM Testing](https://gorm.io/docs/testing.html)
- [WebSocket Testing](https://github.com/gorilla/websocket)

## Contact

For questions about testing or to report issues:
- Open an issue on GitHub
- Contact the development team
- Review test logs in `logs/` directory
