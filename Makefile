.PHONY: build test clean install fmt vet lint check serve

build:
	go build -o twi-map .

test:
	go test ./...

fmt:
	gofmt -w .
	find . -name '*.go' -exec gofmt -w {} +

vet:
	go vet ./...

lint: vet
	@echo "Run 'go install honnef.co/go/tools/cmd/staticcheck@latest' if not installed"
	-staticcheck ./...

check: fmt vet test

clean:
	rm -f twi-map

install:
	go install .

serve: build
	./twi-map serve --addr localhost:8090
