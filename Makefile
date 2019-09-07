# go option
GO         ?= go
PKG        := go mod vendor
LDFLAGS    := -w -s
GOFLAGS    :=
TAGS       := 
BINDIR     := $(CURDIR)/bin
PKGDIR     := github.com/tritonmedia/identifier
CGO_ENABLED := 1
CGO_CFLAGS_ALLOW := -Xpreprocessor

# Required for globs to work correctly
SHELL=/bin/bash


.PHONY: all
all: build

.PHONY: dep
dep:
	@echo " ===> Installing dependencies via '$$(awk '{ print $$1 }' <<< "$(PKG)" )' <=== "
	@CGO_CFLAGS_ALLOW=$(CGO_CFLAGS_ALLOW) $(PKG)

.PHONY: build
build:
	@echo " ===> building releases in ./bin/... <=== "
	CGO_CFLAGS_ALLOW=$(CGO_CFLAGS_ALLOW) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/identifier -v $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKGDIR)

.PHONY: gofmt
gofmt:
	@echo " ===> Running go fmt <==="
	gofmt -w ./

# Test publising messages
.PHONY: test-v1.identify-publish
test-v1.identify-publish:
	go run ./test/v1.identify-publish.go

.PHONY: test-v1.identify.newfile-publish
test-v1.identify.newfile-publish:
	go run ./test/v1.identify.newfile-publish.go

.PHONY: update-schemas
update-schema:
	@rm $(CURDIR)/pkg/storageapi/postgres/schema/schema.go || true
	go-embed -input $(CURDIR)/pkg/storageapi/postgres/schema -output $(CURDIR)/schema.go
	@mv $(CURDIR)/schema.go $(CURDIR)/pkg/storageapi/postgres/schema/schema.go