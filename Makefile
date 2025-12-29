ROOT_DIR    = $(shell pwd)
NAMESPACE   = "default"
DEPLOY_NAME = "template-single"
DOCKER_NAME = "template-single"

.PHONY: generate-config
generate-config:
	@if [ -f "./scripts/generate-config.sh" ]; then \
		./scripts/generate-config.sh; \
	else \
		echo "Template file not found, skipping config generation"; \
	fi

include ./hack/hack-cli.mk
include ./hack/hack.mk