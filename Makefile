build:
	@go build -o bin/yoPoker

run: build 
	@./bin/yoPoker

test:
	go test -v ./...