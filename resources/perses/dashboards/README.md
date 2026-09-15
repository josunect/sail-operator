# Istio Perses dashboards

Vendored `PersesDashboard` manifests from [perses/community-mixins](https://github.com/perses/community-mixins).

| Field | Value |
|-------|-------|
| Source | `examples/dashboards/operator/istio/` |
| Ref | `v0.7.0` |
| Last updated | `2026-09-15` |

## Bundled dashboards

| Dashboard ID | Display name |
|--------------|--------------|
| `istio-control-plane` | Istio Control Plane Dashboard |
| `istio-mesh-dashboard` | Istio Mesh Dashboard |
| `istio-performance` | Istio Performance Dashboard |
| `istio-service-dashboard` | Istio Service Dashboard |
| `istio-workload-dashboard` | Istio Workload Dashboard |
| `istio-ztunnel-dashboard` | Istio Ztunnel Dashboard |

Dashboard IDs must remain stable for Kiali and other consumers.

## PersesDatasource requirement

The Sail Operator does **not** create or manage `PersesDatasource` resources. You must create one manually in the target namespace (`sailoperator.io/perses-project`) before the dashboards can display data.

**The datasource must be named `prometheus-datasource`.** All panels in these dashboards reference that name. If your `PersesDatasource` uses a different name, the dashboards will be created but queries will not resolve and panels will show no data.

Example:

```yaml
apiVersion: perses.dev/v1alpha2
kind: PersesDatasource
metadata:
  name: prometheus-datasource   # required name
  namespace: monitoring         # must match sailoperator.io/perses-project
spec:
  # configure your Prometheus/Thanos endpoint
```

## Enabling dashboard provisioning

Annotate the `Istio` CR:

```yaml
apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: default
  annotations:
    sailoperator.io/perses-dashboards: enabled
    sailoperator.io/perses-project: monitoring
```

Regenerate these files with `hack/update-perses-dashboards.sh [COMMUNITY_MIXINS_REF]`.
