#!/bin/bash

# Update package index
sudo yum update -y

# Install Docker
sudo yum install -y docker

# Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Enable remote access to Docker
sudo mkdir -p /etc/systemd/system/docker.service.d/
cat <<EOF | sudo tee /etc/systemd/system/docker.service.d/override.conf
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H tcp://0.0.0.0:2375
EOF

# Reload systemd and restart Docker
sudo systemctl daemon-reload
sudo systemctl restart docker

echo "Docker installed and configured for remote access."

