.PHONY: all
all: mutating-webhook

.PHONY: bin
bin:
	if ! [ -d bin ]; then mkdir bin; fi;

.PHONY: mutating-webhook
mutating-webhook: bin
	go build -o bin/mutating-webhook

.PHONY: docker
docker:
	docker build -t mauricethomas/goapp-mutating-webhook:latest .

.PHONY: kind
kind:
	kind create cluster

.PHONY: load-image
load-image:
	kind load docker-image mauricethomas/goapp-mutating-webhook:latest

.PHONY: apply
apply:
	kubectl apply -f deployment

.PHONY: cert-manager
cert-manager:
	kubectl config use-context kind-kind
	helm repo add jetstack https://charts.jetstack.io --force-update
	helm repo update
	helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.17.0 --set crds.enabled=true

.PHONY: prometheus
prometheus:
	kubectl config use-context kind-kind
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm install prometheus --namespace prometheus --create-namespace prometheus-community/kube-prometheus-stack
