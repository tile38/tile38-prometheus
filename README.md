# tile38-prometheus

This repository provides a service that connects to a [Tile38](https://github.com/tidwall/tile38) instance
and exposes an endpoint that delivers [Prometheus](https://prometheus.io) compatible metrics.

## Getting Started

### Docker

Perhaps the easiest way to get up and running is with Docker.

```
docker run -e TILE38_ADDR=192.168.7.87:9851 -p 8080:8080 tile38/tile38-prometheus
```

This will start the `tile38-prometheus` service and it to a Tile38 instance at 192.168.7.87:9851.  
You can now see the metrics output at http://localhost:8080/metrics.

### Building

[Go](https://golang.org) must be installed on the build machine.

To build everything:

```
$ make
```

### Running

For command line options invoke:

```
$ ./tile38-prometheus -h
```

To run the service and connect it to a Tile38 instance at localhost:9851:

```
$ ./tile38-prometheus --tile38-addr localhost:9851
```

Optionally define a namespace for your metrics via:

```
$ ./tile38-prometheus --tile38-addr localhost:9851 --namespace myservice
```

You can now see the metrics output at http://localhost:8080/metrics.

## License

Source code is available under the [MIT License](/LICENSE).
