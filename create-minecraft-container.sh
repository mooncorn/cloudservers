#!/bin/bash

# Define variables
SERVER_NAME="my-minecraft-server"
MINECRAFT_VERSION="latest"  # You can specify a specific Minecraft version, e.g., "1.17.1"
MEMORY="2G"  # Amount of memory to allocate to the server

# Pull the latest itzg/minecraft-server Docker image
sudo docker pull itzg/minecraft-server:latest

# Create a directory to store server files
sudo mkdir -p $SERVER_NAME
sudo cd $SERVER_NAME

# Run the Minecraft server container
sudo docker run -d --name $SERVER_NAME \
    -e EULA=TRUE \
    -e VERSION=$MINECRAFT_VERSION \
    -e TYPE=SPIGOT \
    -p 25565:25565 \
    -e MEMORY=$MEMORY \
    -v $(pwd):/data \
    itzg/minecraft-server:latest

# Display server information
echo "Minecraft server $SERVER_NAME has been created and is running."
echo "You can connect to it using the IP address of your host machine."
