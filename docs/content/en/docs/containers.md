---
title: "Containerized Deployments"
linkTitle: "Containers"
weight: 2
description: >
  Deploy OpenSOHO using Docker, Podman, or Kubernetes.
---

## Docker / Podman

Prebuilt OpenSOHO container images are available at
[github.com/orgs/opensoho/packages](https://github.com/orgs/opensoho/packages).

A custom container image can be built using the `Dockerfile` in the repository.

### Docker Compose

```yaml
opensoho:
  image: ghcr.io/opensoho/opensoho:v0.9.0
  container_name: opensoho
  command: serve --http 0.0.0.0:8090
  environment:
    - OPENSOHO_SHARED_SECRET=LoNgExAmPleStrInGoF32cHarActeRs5
  volumes:
    - "./opensoho/pb_data:/ko-app/pb_data"
  ports:
    - 8090:8090
  restart: unless-stopped
```

Mounting a volume to `/ko-app/pb_data` preserves your configuration and history across upgrades.
Copy the contents of the `pb_data` directory when migrating.

## Kubernetes

A Helm chart is available, provided by the community.
