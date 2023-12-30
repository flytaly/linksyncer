make:
	go run main.go

test:
	go test --tags=unittest ./...

build:
	go build -o imagesync main.go

install:
	go install
