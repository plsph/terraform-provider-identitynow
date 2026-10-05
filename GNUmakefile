default: build

build:
	@go build -o /dev/null ./...

install:
	@go install -v ./...

test:
	@go test ./...

testrace:
	@go test -race ./...

vet:
	@go vet ./...

fmt:
	@gofmt -w .

fmtcheck:
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:"; gofmt -l .; exit 1)

generate:
	@cd tools && go generate ./...

testacc:
	@sh -c "'$(CURDIR)/scripts/gotestacc.sh'"

.PHONY: default build install test testrace vet fmt fmtcheck generate testacc
