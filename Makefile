.PHONY: tidy lint fmt

tidy:
	go mod tidy

lint: 
	golangci-lint run
	staticcheck ./...

fmt:
	gofumpt -w .
	goimports -w .
	gci write --skip-generated -s 'standard' -s 'default' -s 'prefix(github.com/hiteshrepo)' .
