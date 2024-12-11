#!/bin/bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib
sudo -u postgres psql -c "CREATE USER validator WITH PASSWORD 'val1dat0r';"
sudo -u postgres psql -c "CREATE DATABASE project-sem-1;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE project-sem-1 TO validator;"
sudo systemctl restart postgresql