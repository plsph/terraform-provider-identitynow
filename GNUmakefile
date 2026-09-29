default: build

build:
	@go build -o /dev/null ./...

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

testacc:
	@sh -c "'$(CURDIR)/scripts/gotestacc.sh'"

.PHONY: default build test testrace vet fmt fmtcheck testacc
