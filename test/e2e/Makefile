.PHONY: all

# cluster setup
e2e-setup:
	@echo "Setting up e2e test cluster"
	./script/start_minikube_cluster.sh up -c e2e-test

e2e-destroy:
	@echo "Destroying e2e test cluster"
	./script/start_minikube_cluster.sh down -c e2e-test

# download ginkgo
PROJECT_DIR := $(shell dirname $(shell dirname $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))))
GINKGO=$(PROJECT_DIR)/bin/ginkgo
.PHONY: ginkgo
ginkgo: ## Download ginkgo locally if necessary.
	@echo "Downloading ginkgo"
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo@v1.16.5)

# run e2e tests
e2e-test:
ifeq ($(mode), standard)
	$(GINKGO) -v ./standard/
else
	@echo "Invalid mode: $(mode)"
endif


# utils
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2); \
}
endef