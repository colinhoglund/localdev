# localdev

Setup a kind cluster for local development on macOS.

## Requirements
- [go](https://go.dev/doc/install)
- [podman](https://podman.io/docs/installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/)

## Usage

Install a podman VM
```
make podman-init
```

Configure Kind cluster (requires sudo password to add /etc/resolver/example.com)
```
make kind-all
```

Verify local DNS is working
```
curl -i http://registry.example.com/v2/
```
