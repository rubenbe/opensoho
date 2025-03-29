# Image names
SERVER_IMAGE=opensoho/server
CLIENT_IMAGE=opensoho/docker

# Default target
all: build run

# Build both images
build: build-server build-client

# Build the PocketBase server image
build-server: Dockerfile.pocketbase
	podman build -t $(SERVER_IMAGE) -f Dockerfile.pocketbase

# Build the client image
build-client: openwrt_docker/Dockerfile
	podman build -t $(CLIENT_IMAGE) -f openwrt_docker/Dockerfile

# Deploy the pod
run: build
	podman play kube podman-kube.yaml

# Stop the pod
stop:
	podman pod rm -f opensoho-pod || true

# Clean everything
clean: stop
	podman rmi -f $(SERVER_IMAGE) $(CLIENT_IMAGE) || true

