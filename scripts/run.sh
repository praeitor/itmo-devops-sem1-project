#!/bin/bash
ssh user@<server_ip> << 'EOF'
sudo apt update
sudo apt install -y golang git postgresql
git clone <repo_url>
cd <repo_folder>
go build -o app main.go
nohup ./app &
EOF