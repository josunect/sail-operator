# Istio Perses dashboards (productized)

Vendored `PersesDashboard` manifests from [perses/community-mixins](https://github.com/perses/community-mixins/tree/main/examples/dashboards/operator/istio).

Bundled files:

- `istio-control-plane.yaml`
- `istio-mesh-dashboard.yaml`
- `istio-performance.yaml`
- `istio-service-dashboard.yaml`
- `istio-workload-dashboard.yaml`
- `istio-ztunnel-dashboard.yaml`

The Integrations controller embeds `pkg/perses/dashboards/*.yaml` at build time.

Each manifest must define `spec.config`. `TestBundledDashboardsHaveSpecConfig` enforces this in unit tests.

For local end-to-end testing on KinD, see `docs/integrations/metrics-integration-perses.adoc`.
