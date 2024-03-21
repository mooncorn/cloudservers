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

# Wait for Docker to restart and become available (not sure if needed)
MAX_ATTEMPTS=30
WAIT_SECONDS=5
attempt=0
while [ $attempt -lt $MAX_ATTEMPTS ]; do
    echo "Attempting to connect to Docker (Attempt $((attempt + 1))/$MAX_ATTEMPTS)..."
    if sudo docker info &>/dev/null; then
        echo "Docker is now ready."
        break
    fi
    attempt=$((attempt + 1))
    sleep $WAIT_SECONDS
done

if [ $attempt -eq $MAX_ATTEMPTS ]; then
    echo "Failed to connect to Docker after $MAX_ATTEMPTS attempts."
    exit 1
fi

echo "Docker installed and configured for remote access."

