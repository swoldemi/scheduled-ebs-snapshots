GOBIN := $(GOPATH)/bin
GOIMPORTS := $(GOBIN)/goimports
GOLANGCILINT := $(GOBIN)/golangci-lint
GOREPORTCARDCLI := $(GOBIN)/goreportcard-cli
GOMETALINTER := $(GOBIN)/gometalinter

# Rules for tooling binaries
$(GOIMPORTS):
	go install golang.org/x/tools/cmd/goimports
$(GOLANGCILINT):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
$(GOREPORTCARDCLI):
	go install github.com/gojp/goreportcard/cmd/goreportcard-cli
$(GOMETALINTER):
	curl -L https://git.io/vp6lP | bash -s -- -b $(GOBIN)

# Static code analysis tooling and checks
.PHONY: check
check: setup
	goimports -w -l -e .
	golangci-lint run ./... \
		-E goconst \
		-E gocyclo \
		-E gosec  \
		-E gofmt \
		-E maligned \
		-E misspell \
		-E nakedret \
		-E unconvert \
		-E unparam \
		-E dupl
	goreportcard-cli -v -t 90

.PHONY: setup
setup: $(GOIMPORTS) $(GOLANGCILINT) $(GOREPORTCARDCLI) $(GOMETALINTER) $(PROTOTOOL)
