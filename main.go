package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/gjson"
)

// metrics is the slice of all data desired in the metrics output
var metrics = []metric{
	// Go/Memory Stats
	metric{"gauge", "go_goroutines", "Number of goroutines that currently exist"},
	metric{"gauge", "go_threads", "Number of OS threads created"},
	metric{"gauge", "alloc_bytes", "Number of bytes allocated and still in use"},
	metric{"counter", "alloc_bytes_total", "Total number of bytes allocated, even if freed"},
	metric{"gauge", "sys_cpus", "Number of CPUS available on the system"},
	metric{"gauge", "sys_bytes", "Number of bytes obtained from system"},
	metric{"counter", "lookups_total", "Total number of pointer lookups"},
	metric{"counter", "mallocs_total", "Total number of mallocs"},
	metric{"counter", "frees_total", "Total number of frees"},
	metric{"gauge", "heap_alloc_bytes", "Number of heap bytes allocated and still in use"},
	metric{"gauge", "heap_sys_bytes", "Number of heap bytes obtained from system"},
	metric{"gauge", "heap_idle_bytes", "Number of heap bytes waiting to be used"},
	metric{"gauge", "heap_inuse_bytes", "Number of heap bytes that are in use"},
	metric{"gauge", "heap_released_bytes", "Number of heap bytes released to OS"},
	metric{"gauge", "heap_objects", "Number of allocated objects"},
	metric{"gauge", "stack_inuse_bytes", "Number of bytes in use by the stack allocator"},
	metric{"gauge", "stack_sys_bytes", "Number of bytes obtained from system for stack allocator"},
	metric{"gauge", "mspan_inuse_bytes", "Number of bytes in use by mspan structures"},
	metric{"gauge", "mspan_sys_bytes", "Number of bytes used for mspan structures obtained from system"},
	metric{"gauge", "mcache_inuse_bytes", "Number of bytes in use by mcache structures"},
	metric{"gauge", "mcache_sys_bytes", "Number of bytes used for mcache structures obtained from system"},
	metric{"gauge", "buck_hash_sys_bytes", "Number of bytes used by the profiling bucket hash table"},
	metric{"gauge", "gc_sys_bytes", "Number of bytes used for garbage collection system metadata"},
	metric{"gauge", "other_sys_bytes", "Number of bytes used for other system allocations"},
	metric{"gauge", "next_gc_bytes", "Number of heap bytes when next garbage collection will take place"},
	metric{"gauge", "last_gc_time_seconds", "Number of seconds since 1970 of last garbage collection"},
	metric{"gauge", "gc_cpu_fraction", "The fraction of this program's available CPU time used by the GC since the program started"},
	// Tile38 Stats
	metric{"gauge", "tile38_pid", "The process ID of the server"},
	metric{"gauge", "tile38_max_heap_size", "Maximum heap size allowed"},
	metric{"gauge", "tile38_read_only", "Whether or not the server is read-only"},
	metric{"gauge", "tile38_pointer_size", "Size of pointer"},
	metric{"counter", "tile38_uptime_in_seconds", "Uptime of the Tile38 server in seconds"},
	metric{"gauge", "tile38_connected_clients", "Number of currently connected Tile38 clients"},
	metric{"gauge", "tile38_cluster_enabled", "Whether or not a cluster is enabled"},
	metric{"gauge", "tile38_aof_enabled", "Whether or not the Tile38 AOF is enabled"},
	metric{"gauge", "tile38_aof_rewrite_in_progress", "Whether or not an AOF shrink is currently in progress"},
	metric{"gauge", "tile38_aof_last_rewrite_time_sec", "Length of time the last AOF shrink took"},
	metric{"gauge", "tile38_aof_current_rewrite_time_sec", "Duration of the on-going AOF rewrite operation if any"},
	metric{"gauge", "tile38_aof_size", "Total size of the AOF in bytes"},
	metric{"gauge", "tile38_http_transport", "Whether or no the HTTP transport is being served"},
	metric{"counter", "tile38_total_connections_received", "Number of connections accepted by the server"},
	metric{"counter", "tile38_total_commands_processed", "Number of commands processed by the server"},
	metric{"counter", "tile38_expired_keys", "Number of key expiration events"},
	metric{"gauge", "tile38_connected_slaves", "Number of connected slaves"},
	metric{"gauge", "tile38_num_points", "Number of points in the database"},
	metric{"gauge", "tile38_num_objects", "Number of objects in the database"},
	metric{"gauge", "tile38_num_strings", "Number of string in the database"},
	metric{"gauge", "tile38_num_collections", "Number of collections in the database"},
	metric{"gauge", "tile38_num_hooks", "Number of hooks in the database"},
	metric{"gauge", "tile38_avg_point_size", "Average point size in bytes"},
	metric{"gauge", "tile38_in_memory_size", "Total in memory size of all collections"},
}

var pool *redis.Pool

func main() {
	var tile38Auth string
	var tile38Addr string
	var httpAddr string
	var namespace string

	flag.StringVar(&tile38Auth, "tile38-auth", "", "tile38 auth")
	flag.StringVar(&tile38Addr, "tile38-addr", ":9851", "address to tile38 server")
	flag.StringVar(&httpAddr, "http-addr", ":8080", "http server address")
	flag.StringVar(&namespace, "namespace", "", "metrics namespace")

	flag.Usage = func() {
		fmt.Printf("Usage: ./tile38-prometheus [--tile38-addr addr] [options]\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		fmt.Printf("    --tile38-auth auth  : Tile38 AUTH password (default \"\")\n")
		fmt.Printf("    --tile38-addr addr  : Address to Tile38 instance (default \":9851\")\n")
		fmt.Printf("    --http-addr addr    : HTTP server listening address (default \":8080\")\n")
		fmt.Printf("    --namespace namespace    : optional metrics namespace (default \"\")\n")
		fmt.Printf("\n")
		fmt.Printf("Environment variables:\n")
		fmt.Printf("    TILE38_AUTH=<auth>\n")
		fmt.Printf("    TILE38_ADDR=<addr>\n")
		fmt.Printf("\n")
		fmt.Printf("Examples:\n")
		fmt.Printf("    ./tile38-prometheus --tile38-addr 10.43.12.45:9851\n")
		fmt.Printf("    TILE38_ADDR=10.43.12.45:9851 ./tile38-prometheus\n")
		fmt.Printf("\n")
	}
	flag.Parse()
	if v := os.Getenv("TILE38_AUTH"); v != "" {
		tile38Auth = v
	}
	if v := os.Getenv("TILE38_ADDR"); v != "" {
		tile38Addr = v
	}

	// Create the Tile38 connection pooler, which is responsible for
	// maintaining stable connections to the Tile38 server.
	pool = redis.NewPool(func() (redis.Conn, error) {
		conn, err := redis.Dial("tcp", tile38Addr)
		if err != nil {
			return nil, err
		}
		if _, err := do(conn, "OUTPUT", "json"); err != nil {
			conn.Close()
			return nil, err
		}
		if tile38Auth != "" {
			if _, err := do(conn, "auth", tile38Auth); err != nil {
				conn.Close()
				return nil, err
			}
		}
		return conn, nil
	}, 5)

	// create an http HandleFunc that retrieves statistics from Tile38
	// and produces a valid prometheus metrics output.
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, namespace)
	})

	go func() {
		time.Sleep(time.Second)
		log.Printf("Server started at %v", httpAddr)
		log.Printf("Pointing to Tile38 server at %v", tile38Addr)
	}()
	log.Printf("%s", http.ListenAndServe(httpAddr, nil))
}

func handle(w http.ResponseWriter, rd *http.Request, n string) {
	conn := pool.Get()
	defer conn.Close()

	out, err := do(conn, "SERVER", "ext")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	m := gjson.Get(out, "stats").Map()

	// Produce a fully populated prometheus metrics output
	var promOutput string
	for _, metric := range metrics {
		promOutput += metric.promString(get(m, metric.Key), n)
	}

	// Return a fully populated prometheus document
	w.Write([]byte(promOutput))
}

func do(conn redis.Conn, cmd string, args ...interface{}) (string, error) {
	out, err := redis.String(conn.Do(cmd, args...))
	if err != nil {
		return "", err
	}
	if !gjson.Get(out, "ok").Bool() {
		return "", errors.New(gjson.Get(out, "err").String())
	}
	return out, err
}

// metric is a type of struct used to store a metrics type, key and description
type metric struct{ Type, Key, Desc string }

// promString returns the prometheus string representation of the metric given
// a float64 value
func (m metric) promString(val float64, n string) string {
	if len(n) > 0 {
		return fmt.Sprintf("# HELP %s_%s %s\n", n, m.Key, m.Desc) +
			fmt.Sprintf("# TYPE %s_%s %s\n", n, m.Key, m.Type) +
			fmt.Sprintf("%s_%s %s\n", n, m.Key, strconv.FormatFloat(val, 'f', -1, 64))
	}
	return fmt.Sprintf("# HELP %s %s\n", m.Key, m.Desc) +
		fmt.Sprintf("# TYPE %s %s\n", m.Key, m.Type) +
		fmt.Sprintf("%s %s\n", m.Key, strconv.FormatFloat(val, 'f', -1, 64))
}

// get retrieves a value by its passed json key and returns it as a float64. If
// it fails to find the key or fails to assert it to a float64 9999.9999 is
// returned as an obvious error
func get(m map[string]gjson.Result, key string) float64 {
	switch m[key].Type {
	case gjson.True:
		return 1
	case gjson.False:
		return 0
	case gjson.Number:
		return m[key].Num
	default:
		return math.NaN()
	}
}
