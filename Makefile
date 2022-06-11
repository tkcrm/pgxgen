install:
	go install ./cmd/pgxgen

upgrade:
	go-mod-upgrade
	go mod tidy
