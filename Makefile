SHELL := /bin/bash

VERSION := 1.0
KIND_CLUSTER := prolog-cluster
PROLOG_DOCKER_IMAGE := prolog-amd64

tidy:
	go mod tidy && go mod vendor

all: lola-api

run:
	go run app/cli/prolog/main.go api serve

lola-api:
	docker build \
		-f infra/docker/Dockerfile.prolog \
		-t $(PROLOG_DOCKER_IMAGE):$(VERSION) \
		--build-arg VCS_REF=$(VERSION) \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.



# ==============================================================================
# Running from within k8s/kind


kind-up:
	kind create cluster \
		--image kindest/node:v1.22.0 \
		--name $(KIND_CLUSTER) \
		--config infra/k8s/kind/kind-config.yaml
	kubectl config set-context --current --namespace=prolog

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-restart:
	kubectl rollout restart deployment prolog-pod

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-status-prolog:
	kubectl get pods -o wide --watch

kind-load:
	cd infra/k8s/kind/prolog-pod; kustomize edit set image prolog-image=$(PROLOG_DOCKER_IMAGE):$(VERSION)
	kind load docker-image $(PROLOG_DOCKER_IMAGE):$(VERSION) --name $(KIND_CLUSTER)

kind-apply:
	kustomize build infra/k8s/kind/prolog-pod | kubectl apply -f -

kind-describe:
	kubectl describe pod -l app=prolog

kind-logs:
	kubectl logs -l app=prolog --all-containers=true -f --tail=100

kind-service-delete:
	kustomize build infra/k8s/kind/prolog-pod | kubectl delete -f -

