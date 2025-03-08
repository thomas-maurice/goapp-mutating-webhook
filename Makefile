.PHONY: all
all: mutating-webhook

.PHONY: bin
bin:
	if ! [ -d bin ]; then mkdir bin; fi;

.PHONY: mutating-webhook
mutating-webhook: bin
	go build -o bin/mutating-webhook

docker:
	docker build -t mutating-webhook:latest .

kind:
	kind create cluster

load-image:
	kind load docker-image mutating-webhook:latest

apply:
	kubectl apply -f deployment

cert-manager:
	kubectl config use-context kind-kind
	helm repo add jetstack https://charts.jetstack.io --force-update
	helm repo update
	helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.17.0 --set crds.enabled=true