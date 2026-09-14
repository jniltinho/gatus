BINARY=gatus
# Commit do upstream (TwiN/gatus) em que o fork está baseado (após a v5.36.0); atualizar ao sincronizar
UPSTREAM_BASE ?= 7f3873d0b455a0d2bf7880eeee1dd0ea7391b2ab
DOCKER_IMAGE ?= jniltinho/gatus
DOCKER_PLATFORMS ?= linux/amd64,linux/arm64
VERSION ?= dev
DIST := dist
RELEASE_ARCHS := amd64 arm64
# Arquivos Go adicionados ou alterados pelo fork desde UPSTREAM_BASE (inclui os não commitados)
FORK_GO_FILES = $(shell { git diff --name-only --diff-filter=ACMR $(UPSTREAM_BASE) -- '*.go'; git ls-files --others --exclude-standard -- '*.go'; } 2>/dev/null | sort -u)

.PHONY: install
install:
	go build -v -o $(BINARY) .

.PHONY: run
run:
	ENVIRONMENT=dev GATUS_CONFIG_PATH=./config.yaml go run main.go

.PHONY: run-binary
run-binary:
	ENVIRONMENT=dev GATUS_CONFIG_PATH=./config.yaml ./$(BINARY)

.PHONY: clean
clean:
	rm $(BINARY)

.PHONY: test
test:
	go test ./... -cover

.PHONY: build
build:
	@mkdir -p $(DIST)
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST)/$(BINARY) .

.PHONY: check-upstream-base
check-upstream-base:
	@git cat-file -e "$(UPSTREAM_BASE)^{commit}" 2>/dev/null || { echo "Commit $(UPSTREAM_BASE) não encontrado (o clone precisa do histórico completo)"; exit 1; }

# gofmt só nos arquivos do fork, para não reformatar código do upstream
.PHONY: fmt
fmt: check-upstream-base
	@files="$(FORK_GO_FILES)"; [ -z "$$files" ] || gofmt -w $$files

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: check-upstream-base vet
	@files="$(FORK_GO_FILES)"; unformatted=$$([ -z "$$files" ] || gofmt -l $$files); \
	if [ -n "$$unformatted" ]; then echo "Arquivos precisando de gofmt (execute: make fmt):"; echo "$$unformatted"; exit 1; fi

.PHONY: release-cross
release-cross:
	@rm -rf $(DIST)/pkg
	@for arch in $(RELEASE_ARCHS); do \
		mkdir -p $(DIST)/pkg/linux_$$arch && \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -ldflags "-s -w" -o $(DIST)/pkg/linux_$$arch/$(BINARY) . && \
		tar -czf $(DIST)/$(BINARY)_$(VERSION)_linux_$$arch.tar.gz -C $(DIST)/pkg/linux_$$arch $(BINARY) -C $(CURDIR) config.yaml LICENSE README.md && \
		echo "  $(DIST)/$(BINARY)_$(VERSION)_linux_$$arch.tar.gz" || exit 1; \
	done


# Publica a imagem multi-arquitetura no Docker Hub a partir desta máquina (exige docker login)
.PHONY: docker-release
docker-release:
	@[ "$(VERSION)" != "dev" ] || { echo "Defina a versão: make docker-release VERSION=5.36.0-fork.1"; exit 1; }
	$(MAKE) release-cross VERSION=$(VERSION)
	docker buildx build -f Dockerfile.release --platform $(DOCKER_PLATFORMS) -t $(DOCKER_IMAGE):v$(VERSION) --push .

##########
# Docker #
##########

.PHONY: docker-build
docker-build:
	docker build -t jniltinho/gatus:$(VERSION) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 --name gatus jniltinho/gatus:$(VERSION)

.PHONY: docker-build-and-run
docker-build-and-run: docker-build docker-run


#############
# Front end #
#############

.PHONY: frontend-install
frontend-install:
	npm --prefix web/app install

.PHONY: frontend-build
frontend-build:
	npm --prefix web/app run build

.PHONY: frontend-dev
frontend-dev:
	npm --prefix web/app run serve
