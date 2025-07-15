#!/bin/bash

# WindGo Chat Database Setup Script using Podman Compose
# This script manages PostgreSQL database using podman-compose

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if podman-compose is installed
check_podman_compose() {
    if ! command -v podman-compose &> /dev/null; then
        print_error "podman-compose is not installed. Please install it first."
        exit 1
    fi
    print_success "podman-compose is available"
}

# Check if required files exist
check_files() {
    if [ ! -f "docker-compose.yml" ]; then
        print_error "docker-compose.yml not found in current directory"
        exit 1
    fi

    if [ ! -f "init.sql" ]; then
        print_error "init.sql not found in current directory"
        exit 1
    fi

    print_success "Required files found"
}

# Start database services
start_database() {
    print_info "Starting database services using podman-compose..."
    podman-compose up -d postgres
    print_success "PostgreSQL container started"
}

# Start adminer
start_adminer() {
    print_info "Starting Adminer for database management..."
    podman-compose up -d adminer
    print_success "Adminer started"
}

# Wait for PostgreSQL to be ready
wait_for_postgres() {
    print_info "Waiting for PostgreSQL to initialize and be ready..."

    local max_attempts=60
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if podman-compose exec postgres pg_isready -U postgres -d chatapp >/dev/null 2>&1; then
            print_success "PostgreSQL is ready!"
            return 0
        fi

        if [ $((attempt % 10)) -eq 0 ]; then
            print_info "Attempt $attempt/$max_attempts - Still waiting for PostgreSQL..."
        fi

        sleep 2
        ((attempt++))
    done

    print_error "PostgreSQL failed to start within expected time"
    return 1
}

# Test database and show initialization results
test_database() {
    print_info "Testing database connection and showing initialization results..."

    # Test connection
    if podman-compose exec postgres psql -U postgres -d chatapp -c "SELECT version();" >/dev/null 2>&1; then
        print_success "Database connection successful"
    else
        print_error "Database connection failed"
        return 1
    fi

    # Show initialization results
    echo ""
    print_header "Database Initialization Results"
    podman-compose exec postgres psql -U postgres -d chatapp -c "SELECT * FROM check_db_health();"
}

# Show connection information
show_info() {
    echo ""
    print_header "WindGo Chat Database Setup Complete"
    echo ""
    print_success "PostgreSQL Database:"
    echo "  Host: localhost"
    echo "  Port: 5432"
    echo "  Database: chatapp"
    echo "  Username: postgres"
    echo "  Password: password"
    echo ""
    print_success "Adminer (Database Management):"
    echo "  URL: http://localhost:8081"
    echo "  System: PostgreSQL"
    echo "  Server: postgres:5432"
    echo "  Username: postgres"
    echo "  Password: password"
    echo "  Database: chatapp"
    echo ""
    print_success "Demo Accounts (password: admin123):"
    echo "  Admin: admin@windgo.com"
    echo "  Demo User: demo@windgo.com"
    echo ""
    print_info "Container Status:"
    podman-compose ps
}

# Stop services
stop_services() {
    print_info "Stopping database services..."
    podman-compose down
    print_success "Services stopped"
}

# Show logs
show_logs() {
    print_info "Showing PostgreSQL logs (press Ctrl+C to exit)..."
    podman-compose logs -f postgres
}

# Open PostgreSQL shell
open_shell() {
    print_info "Opening PostgreSQL shell..."
    podman-compose exec postgres psql -U postgres -d chatapp
}

# Reset database
reset_database() {
    print_warning "This will remove all database data. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        print_info "Stopping services and removing volumes..."
        podman-compose down -v
        print_success "Database reset complete"
        start_setup
    else
        print_info "Reset cancelled"
    fi
}

# Show container status
show_status() {
    print_header "Container Status"
    podman-compose ps
}

# Create database backup
create_backup() {
    local backup_file="windgo_chat_backup_$(date +%Y%m%d_%H%M%S).sql"
    print_info "Creating database backup: $backup_file"

    if podman-compose ps postgres | grep -q "Up"; then
        podman-compose exec postgres pg_dump -U postgres chatapp > "$backup_file"
        print_success "Backup created: $backup_file"
    else
        print_error "PostgreSQL container is not running"
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start     Start the database services (default)"
    echo "  stop      Stop the database services"
    echo "  restart   Restart the database services"
    echo "  reset     Reset the database (removes all data)"
    echo "  status    Show container status"
    echo "  logs      Show PostgreSQL logs"
    echo "  shell     Open PostgreSQL shell"
    echo "  backup    Create database backup"
    echo "  help      Show this help message"
}

# Main setup function
start_setup() {
    print_header "Setting up WindGo Chat Database"

    check_podman_compose
    check_files
    start_database
    start_adminer
    wait_for_postgres
    test_database
    show_info
}

# Main script logic
case "${1:-start}" in
    start)
        start_setup
        ;;
    stop)
        stop_services
        ;;
    restart)
        stop_services
        sleep 2
        start_setup
        ;;
    reset)
        reset_database
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    shell)
        open_shell
        ;;
    backup)
        create_backup
        ;;
    help)
        show_usage
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_usage
        exit 1
        ;;
esac
