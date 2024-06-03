.PHONY: build
build:
	@echo "Building docker image with docker buildx..."
	@./build.sh

.PHONY: install
install:
	@kubectl apply -f deploy/manifests.yaml

.PHONY: uninstall
uninstall: 
	@kubectl delete -f deploy/manifests.yaml
