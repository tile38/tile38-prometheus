deps:
	-rm Gopkg.toml
	-rm Gopkg.lock
	-rm -r vendor
	dep init
test:
	go clean
	go test ./...
run:
	go run main.go
build:
	-rm tile38-prometheus
	make deps
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tile38-prometheus
push:
	make build
	-docker rmi tile38/tile38-prometheus
	docker build --no-cache -t tile38/tile38-prometheus .
	docker push tile38/tile38-prometheus
	rm -f tile38-prometheus