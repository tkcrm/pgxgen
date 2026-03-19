install:
	go install ./cmd/pgxgen

build:
	go build -v ./cmd/pgxgen

test:
	go test -v ./...

fmt:
	gofumpt -l -w .

lint:
	golangci-lint run
