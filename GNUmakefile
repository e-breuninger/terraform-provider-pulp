.PHONY: default
default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

docker:
	cd docker && docker compose up -d

dockerdown:
	cd docker && docker compose down --volumes

.PHONY: testenv
testenv: dockerdown docker
	sleep 5

test: testenv
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc: testenv
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testacc build install generate docker
