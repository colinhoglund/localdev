# localdev

Setup a [kind](https://kind.sigs.k8s.io/) cluster for local development on macOS.

## Requirements
- [go](https://go.dev/doc/install)
- [podman](https://podman.io/docs/installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/)

## Usage

Initialize a podman VM.
```
make podman-init
```

Configure a kind cluster (requires sudo password to add `/etc/resolver/example.com`).
```
make kind-all
```

Verify local DNS is working.
```
curl -i http://registry.example.com/v2/
```

Verify registry is working.
```
podman pull alpine:latest
podman tag alpine:latest registry.example.com/alpine:latest
podman push --tls-verify=false registry.example.com/alpine:latest

kubectl --kubeconfig ~/.kube/config.kind-kind --context kind-kind run \
  -it --rm --restart=Never --image registry.example.com/alpine:latest alpine-test \
  -- uname
```

Run `make help` to get a list of commands for managing the local cluster.
