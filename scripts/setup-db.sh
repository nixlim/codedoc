#!/bin/bash
set -e

echo "Setting up PostgreSQL database..."

# Start Docker services
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 5

# Create database if it doesn't exist
docker exec -it codedoc_postgres_1 psql -U codedoc -c "SELECT 1" || \
docker exec -it codedoc_postgres_1 createdb -U codedoc codedoc_dev

echo "Database setup complete!"