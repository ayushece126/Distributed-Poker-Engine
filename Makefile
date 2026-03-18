build:
<<<<<<< HEAD
	@go build -o bin/yoPoker

run: build 
	@./bin/yoPoker

test:
	go test -v ./...
=======
	@go build -o bin/ggpoker

run: build 
	@./bin/ggpoker

test:
	go test -v ./...
>>>>>>> 13356f6 (complete working roundtrip)
