#!/bin/bash

set -e

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to start..."
while ! docker-compose exec -T db pg_isready -U postgres -q; do
  sleep 1
done
echo "PostgreSQL started."

# Load the data
echo "Loading test data..."
docker-compose exec -T db psql -U postgres -d postgres < init.sql
echo "Test data loaded successfully." 