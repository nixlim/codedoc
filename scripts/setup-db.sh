#!/bin/bash
set -e

echo "Setting up PostgreSQL database..."

# Start Docker services
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 5

# Create database if it doesn't exist
docker exec -it codedoc-postgres-1 psql -U codedoc -c "SELECT 1" || \
docker exec -it codedoc-postgres-1 createdb -U codedoc codedoc_dev

# Run migrations
echo "Running database migrations..."
for migration in migrations/*.up.sql; do
    if [ -f "$migration" ]; then
        echo "Applying migration: $migration"
        docker exec -i codedoc-postgres-1 psql -U codedoc -d codedoc_dev < "$migration"
    fi
done

echo "Database setup complete!"