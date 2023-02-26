install:
	go install ./cmd/pgxgen

upgrade:
	go-mod-upgrade
	go mod tidy

build:
	go build -v ./cmd/pgxgen

test:
	go test -v ./...

copysqlc:
	go run cmd/copysqlc/main.go