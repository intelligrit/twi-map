.PHONY: build test clean install

build:
	go build -o twi-map .

test:
	go test ./...

clean:
	rm -f twi-map

install:
	go install .
