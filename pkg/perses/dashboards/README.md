# Istio Perses dashboards (productized)

Vendored `PersesDashboard` manifests from [perses/community-mixins](https://github.com/perses/community-mixins/tree/main/examples/dashboards/operator/istio).

Copy the operator-format YAML files here before enabling dashboard reconciliation in CI/e2e:

- `istio-control-plane.yaml`
- `istio-mesh-dashboard.yaml`
- `istio-performance.yaml`
- `istio-service-dashboard.yaml`
- `istio-workload-dashboard.yaml`
- `istio-ztunnel-dashboard.yaml`

The Integrations controller embeds `pkg/perses/dashboards/*.yaml` at build time.
