all: tile38-prometheus

clean: 
	rm -f tile38-prometheus

.PHONY: tile38-prometheus

tile38-prometheus:
	CGO_ENABLED=0 go build -o tile38-prometheus

