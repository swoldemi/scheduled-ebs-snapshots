S3_BUCKET ?= swoldemi-tmp
DEFAULT_VOLUME_ID ?= vol-0b820eaeab56561d7
DEFAULT_REGION ?= us-east-2
DEFAULT_INTERVAL="1"
DEFAULT_INTERVAL_UNIT="minute"
DEFAULT_CROSS_ACCOUNT_ROLE_ARN ?= arn:aws:iam::273450712882:role/SampleRole
DEFAULT_CROSS_ACCOUNT_ROLE_EXTERNAL_ID ?= SampleExternalID
DEFAULT_STACK_NAME ?= lambda-scheduled-ebs-snapshots

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

.PHONY: build
build: clean
	 go build -v -a -installsuffix cgo -tags netgo -ldflags '-w -extldflags "-static"' main.go

.PHONY: test
test: clean
	go test -v -race -timeout 30s -count=1 -coverprofile=profile.out ./...

# Static code analysis tooling and checks
.PHONY: check
check: setup
	goimports -w -l -e .
	gofmt -s -w .
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
setup: $(GOIMPORTS) $(GOLANGCILINT) $(GOREPORTCARDCLI) $(GOMETALINTER)

.PHONY: tmpl
tmpl: 
	cfn-lint template.yaml

.PHONY: sam-package
sam-package:
	@sam package --template-file template.yaml --s3-bucket $(S3_BUCKET) --output-template-file packaged.yaml

.PHONY: sam-deploy
sam-deploy:
	@sam deploy \
	--parameter-overrides Interval=$(DEFAULT_INTERVAL) IntervalUnit=$(DEFAULT_INTERVAL_UNIT) Region=$(DEFAULT_REGION) VolumeID=$(DEFAULT_VOLUME_ID)  \
	--template-file ./packaged.yaml \
	--stack-name $(DEFAULT_STACK_NAME) \
	--capabilities CAPABILITY_IAM
	aws --region $(DEFAULT_REGION) cloudformation describe-stacks --stack-name $(DEFAULT_STACK_NAME) --query 'Stacks[0].Outputs'

.PHONY: stack-describe 
stack-describe:
	aws --region $(DEFAULT_REGION) cloudformation describe-stacks --stack-name $(DEFAULT_STACK_NAME) --query 'Stacks[0].Outputs'

.PHONY: sam-tail-logs
sam-tail-logs:
	@sam logs --name scheduled-ebs-snapshots --tail

.PHONY: redeploy
redeploy: check tmpl test build sam-package sam-deploy

.PHONY: destroy
destroy: clean
	aws --region $(DEFAULT_REGION) cloudformation delete-stack --stack-name $(DEFAULT_STACK_NAME)

.PHONY: update-mod
update-mod:
	go get $(shell go list -f "{{if not (or .Main .Indirect)}}{{.Path}}{{end}}" -m all)
	go mod tidy

.PHONY: clean
clean:
	rm -f main packaged.yaml profile.
