# go option
GO         ?= go
PKG        := go mod vendor
LDFLAGS    := -w -s
GOFLAGS    :=
TAGS       := 
BINDIR     := $(CURDIR)/bin
PKGDIR     := github.com/tritonmedia/identifier

# Required for globs to work correctly
SHELL=/bin/bash


.PHONY: all
all: build

.PHONY: dep
dep:
	@echo " ===> Installing dependencies via '$$(awk '{ print $$1 }' <<< "$(PKG)" )' <=== "
	@$(PKG)

.PHONY: build
build:
	@echo " ===> building releases in ./bin/... <=== "
	CGO_ENABLED=1 $(GO) build -o $(BINDIR)/identifier -v $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKGDIR)

.PHONY: gofmt
gofmt:
	@echo " ===> Running go fmt <==="
	gofmt -w ./

# Test publising messages
.PHONY: test-publish
test-publish:
	go run ./test/publish.go

.PHONY: update-schemas
update-schema:
	@rm $(CURDIR)/pkg/storageapi/postgres/schema/schema.go || true
	go-embed -input $(CURDIR)/pkg/storageapi/postgres/schema -output $(CURDIR)/schema.go
	@mv $(CURDIR)/schema.go $(CURDIR)/pkg/storageapi/postgres/schema/schema.go