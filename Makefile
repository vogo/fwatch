format:
		goimports -w -l .
		go fmt ./...
		go fix ./...
		gofumpt -w .

license-check:
	# go install github.com/lsm-dev/license-header-checker/cmd/license-header-checker@latest
	license-header-checker -v -a -r apache-license.txt . go

check: license-check
		golangci-lint run

test:
		go test -coverprofile=coverage.out -covermode=atomic ./... -v

build: format check test
