# cloudservers

Cloudservers allows hosting of various docker images.

Manages EC2 instances using docker remote access.

## Getting Started

1. Install Go
2. Install project dependancies 
3. Add ssh key.

```
ssh-add ~/.ssh/your-private-key
```

2. Allow reusing a SSH connection for multiple invocations of the docker CLI.

```
echo -e "ControlMaster auto\nControlPath ~/.ssh/control-%C\nControlPersist yes" >> ~/.ssh/config
```

## Instance Setup

Once the instance is created, it must go through the setup process before it is ready to host docker containers. This setup assumes that the instance is running Amazon Linux 2.

1. Update package index.

```
sudo yum update -y
```

2. Install Docker.

```
sudo yum install -y docker
```

3. Start and enable Docker service.

```
sudo systemctl start docker
sudo systemctl enable docker
```

4. Enable remote access to Docker.

```
sudo mkdir -p /etc/systemd/system/docker.service.d/
cat <<EOF | sudo tee /etc/systemd/system/docker.service.d/override.conf
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// -H tcp://0.0.0.0:2375
EOF
```

5. Reload systemd and restart Docker.

```
sudo systemctl daemon-reload
sudo systemctl restart docker
```

6. (optional) Enable non-root privilages on host. This is required to execute docker commands remotely.

```
sudo groupadd docker
sudo usermod -aG docker $USER
newgrp docker
```

## Connecting to remote host

1. To remotely connect to docker daemon through SSH, setup SSH forwarding. Once forwarded, docker commands will execute on the remote host.

```
ssh -L 2375:localhost:2375 ec2-user@54.91.26.120
```
