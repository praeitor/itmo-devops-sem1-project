#!/bin/bash

echo "Starting server..."
go run main.go > server.log 2>&1 &

# Проверка доступности сервера
for i in {1..10}; do
    if curl -s http://localhost:8080 > /dev/null; then
        echo "Server is ready for work"
        exit 0
    fi
    echo "Attempt $i: Server not ready..."
    sleep 5
done

echo "Error: Server failed to start"
exit 1