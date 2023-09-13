KUBEVERSION ?= v1.24.15

HELMCMD = helm --kubeconfig ~/.kube/config.kind-kind --kube-context kind-kind
KUBECTLCMD = kubectl --kubeconfig ~/.kube/config.kind-kind --context kind-kind

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install localdev CLI
	go install ./cmd/localdev/

## podman targets
.PHONY: podman-delete
podman-delete: ## Destroy the podman VM
	podman machine stop
	podman machine rm --force

.PHONY: podman
podman-init: ## Initialize a new podman VM
	podman machine init --cpus 4 --memory 10192 --rootful --now
	cat config/kind/insecure-registry.config | podman machine ssh "cat > /etc/containers/registries.conf.d/registry-example-com.conf"
	# sets the max_map_count necessary for using Elasticsearch
	podman machine ssh 'sysctl -w vm.max_map_count=262144'

## kind targets
.PHONY: kind
kind-all: kind-dns install kind-delete kind-start service-coredns service-registry service-nginx ## Destroy and re-create the kind cluster and all services

.PHONY: kind-dns
kind-dns: ## Configure example.com DNS resolver
	printf "nameserver 127.0.0.1\nport 30053\n" | sudo tee /etc/resolver/example.com

.PHONY: kind-restart
kind-restart: ## Restart a stopped kind cluster
	podman restart kind-control-plane

.PHONY: kind-start
kind-start: ## Start the kind cluster without services
	localdev kind start --k8s-version=$(KUBEVERSION) --config-file ./cluster.yaml

.PHONY: kind-delete
kind-delete: ## Destroy the kind cluster
	localdev kind delete

## k8s targets
.PHONY: service-coredns
service-coredns: ## Install example.com coredns service in the kind cluster
	helm repo add coredns https://coredns.github.io/helm
	$(HELMCMD) install \
		--namespace kube-system \
		--values ./values/coredns.yaml \
		--version 1.21.0 \
		coredns-example-com \
		coredns/coredns
	localdev kind patch-coredns kube-system coredns-example-com-coredns

.PHONY: service-registry
service-registry: ## Install a registry service in the kind cluster
	helm repo add twunio https://helm.twun.io
	$(HELMCMD) install \
		--namespace kube-system \
		--values ./values/registry.yaml \
		--version 2.2.2 \
		registry \
		twunio/docker-registry

.PHONY: service-nginx
service-nginx: ## Install the nginx ingress controller service in the kind cluster
	$(KUBECTLCMD) apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
