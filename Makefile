make:
	go run main.go

test:
	go test --tags=unittest ./...

build:
	go build -o linksyncer main.go

install:
	go install
