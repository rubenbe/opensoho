# Containerized deployments


## Docker / podman

Prebuilt OpenSOHO container images can be found in https://github.com/orgs/opensoho/packages

A custom container image can be built using the Dockerfile in this repo.

### docker compose

A docker compose setup could be used following this example:

```
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

Mounting a volume to `/ko-app/pb_data` allows to maintain your configuration and history over upgrades or migration when copying the content of the `pb_data` directory. 

## Kubernetes

A Helm chart is available, provided by the community.
