# OpenSoho Helm Chart

This Helm chart deploys OpenSoho, an OpenWrt management platform built on PocketBase.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- A container registry with the OpenSoho image

## Installing the Chart

To install the chart with the release name `my-opensoho`:

```bash
helm install my-opensoho ./helm/opensoho
```

The command deploys OpenSoho on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-opensoho` deployment:

```bash
helm delete my-opensoho
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `nameOverride`            | String to partially override opensoho.fullname | `""`  |
| `fullnameOverride`        | String to fully override opensoho.fullname     | `""`  |

### Image parameters

| Name                | Description                                                                 | Value          |
| ------------------- | --------------------------------------------------------------------------- | -------------- |
| `image.repository`  | OpenSoho image repository                                                   | `opensoho`     |
| `image.tag`         | OpenSoho image tag (immutable tags are recommended)                        | `""`           |
| `image.pullPolicy`  | OpenSoho image pull policy                                                  | `IfNotPresent` |

### Deployment parameters

| Name                                    | Description                                                                                                                      | Value           |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | --------------- |
| `replicaCount`                          | Number of OpenSoho replicas to deploy                                                                                           | `1`             |
| `podAnnotations`                        | Annotations for OpenSoho pods                                                                                                   | `{}`            |
| `podSecurityContext`                    | Security context for OpenSoho pods                                                                                              | `{}`            |
| `securityContext`                       | Security context for OpenSoho containers                                                                                        | `{}`            |
| `serviceAccount.create`                 | Specifies whether a service account should be created                                                                            | `true`          |
| `serviceAccount.annotations`            | Annotations to add to the service account                                                                                        | `{}`            |
| `serviceAccount.name`                   | The name of the service account to use                                                                                          | `""`            |

### Service parameters

| Name               | Description                                                                 | Value       |
| ------------------ | --------------------------------------------------------------------------- | ----------- |
| `service.type`     | OpenSoho service type                                                       | `ClusterIP` |
| `service.port`     | OpenSoho service port                                                       | `8090`      |
| `service.targetPort` | OpenSoho service target port                                             | `8090`      |

### Ingress parameters

| Name                       | Description                                                                                                                      | Value                    |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| `ingress.enabled`          | Enable ingress record generation for OpenSoho                                                                                   | `false`                  |
| `ingress.className`        | IngressClass resource. The value depends on your cluster setup.                                                                 | `""`                     |
| `ingress.annotations`      | Additional annotations for the Ingress resource. To enable certificate autogeneration, place here your cert-manager annotations. | `{}`                     |
| `ingress.hosts`            | An array of hosts to be covered with the ingress record.                                                                        | `[{"host":"opensoho.local","paths":[{"path":"/","pathType":"Prefix"}]}]` |
| `ingress.tls`              | TLS configuration for additional hostname(s) to be covered with this ingress record.                                           | `[]`                     |

### Resource parameters

| Name                       | Description                                                                                                                      | Value   |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `resources.limits`         | The resources limits for the OpenSoho containers                                                                                | `{}`    |
| `resources.requests`       | The requested resources for the OpenSoho containers                                                                             | `{}`    |
| `autoscaling.enabled`      | Enable Horizontal POD autoscaling for OpenSoho                                                                                 | `false` |
| `autoscaling.minReplicas`  | Minimum number of OpenSoho replicas                                                                                             | `1`     |
| `autoscaling.maxReplicas`  | Maximum number of OpenSoho replicas                                                                                             | `100`   |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization percentage                                                                                    | `80`    |

### Storage parameters

| Name                        | Description                                                                                                                      | Value   |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `persistence.enabled`       | Enable persistence using Persistent Volume Claims                                                                               | `true`  |
| `persistence.storageClass`  | Persistent Volume storage class                                                                                                 | `""`    |
| `persistence.accessMode`    | Persistent Volume access mode                                                                                                   | `ReadWriteOnce` |
| `persistence.size`          | Persistent Volume size                                                                                                          | `10Gi`  |
| `persistence.mountPath`     | Mount path for persistent data                                                                                                  | `/app/pb_data` |

### Environment parameters

| Name                        | Description                                                                                                                      | Value   |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `env.OPENSOHO_SHARED_SECRET` | OpenSoho shared secret for authentication                                                                                      | `testtest` |

### Health check parameters

| Name                        | Description                                                                                                                      | Value   |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `healthCheck.enabled`       | Enable readiness probe                                                                                                          | `true`  |
| `livenessProbe.enabled`     | Enable liveness probe                                                                                                           | `true`  |

## Configuration and installation details

### Additional environment variables

You can add more environment variables using the `env` section in `values.yaml`:

```yaml
env:
  OPENSOHO_SHARED_SECRET: "your-secret-here"
  # Add other environment variables as needed
  CUSTOM_VAR: "value"
```

### Persistence

The chart mounts a [Persistent Volume](http://kubernetes.io/docs/user-guide/persistent-volumes/) at the `/app/pb_data` path. The volume is created using dynamic volume provisioning.

### Ingress

This chart provides support for Ingress resources. If you have an available Ingress Controller such as [nginx-ingress](https://kubeapps.com/charts/stable/nginx-ingress) or [traefik](https://kubeapps.com/charts/stable/traefik) installed in your cluster, you can enable the ingress functionality by setting the `ingress.enabled` parameter to `true`.

## Examples

### Basic installation

```bash
helm install my-opensoho ./helm/opensoho
```

### Installation with custom values

```bash
helm install my-opensoho ./helm/opensoho \
  --set image.tag=v1.0.0 \
  --set env.OPENSOHO_SHARED_SECRET=my-secret \
  --set service.type=LoadBalancer \
  --set ingress.enabled=true
```

### Installation with custom values file

```bash
helm install my-opensoho ./helm/opensoho -f my-values.yaml
```

## Troubleshooting

### Check pod status

```bash
kubectl get pods -l app.kubernetes.io/name=opensoho
```

### View logs

```bash
kubectl logs -l app.kubernetes.io/name=opensoho
```

### Check service

```bash
kubectl get svc -l app.kubernetes.io/name=opensoho
```

## License

This chart is licensed under the same license as the OpenSoho project.

