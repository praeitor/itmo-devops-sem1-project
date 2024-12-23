#!/bin/bash
sudo apt update
sudo apt install -y postgresql golang
sudo systemctl start postgresql
sudo -u postgres psql -c "CREATE USER validator WITH PASSWORD 'val1dat0r';"
sudo -u postgres psql -c "CREATE DATABASE project-sem-1;"
sudo -u postgres psql -d project-sem-1 -c "GRANT ALL PRIVILEGES ON DATABASE project-sem-1 TO validator;"
sudo -u postgres psql -d project-sem-1 -c "CREATE TABLE prices (id SERIAL PRIMARY KEY, name TEXT, category TEXT, price NUMERIC(10,2), create_date DATE);"