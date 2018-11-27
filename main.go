package main

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/tidwall/gjson"
	common "github.com/tile38/tile38-cloud-common"
	"github.com/tile38/tile38-cloud-common/tile38"
)

// statsInterval defines how often to retrieve new Tile38 statistics
const statsInterval = time.Second

var (
	// tile38Addr is the IP Address of the Tile38 database
	tile38Addr = common.GetEnv("TILE38_ADDR", "209.97.150.193:9851")

	// metrics is the slice of all data desired in the metrics output
	metrics = []metric{
		// Go/Memory Stats
		metric{"gauge", "go_goroutines", "Number of goroutines that currently exist"},
		metric{"gauge", "go_threads", "Number of OS threads created"},
		// metric{"gauge", "go_version", "A summary of the GC invocation durations"},
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
		// metric{"gauge", "tile38_id", "ID of the server"},
		metric{"gauge", "tile38_pid", "The process ID of the server"},
		// metric{"gauge", "tile38_version", "Version of Tile38 running"},
		metric{"gauge", "tile38_max_heap_size", "Maximum heap size allowed"},
		// metric{"gauge", "tile38_type", "Type of instance running"},
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
)

func main() {
	// Create a pool of new Tile38 connections for use with retrieving extended
	// Tile38 server metrics
	tc := tile38.New(tile38Addr)
	defer tc.Close()

	// Create a new echo instance
	e := echo.New()
	e.HideBanner = true
	e.GET("/metrics", metricsHandler(tc))
	e.Logger.Fatal(e.Start(":8080"))
}

// metricsHandler is an echo HandlerFunc that retrieves statistics from Tile38 and
// produces a valid prometheus metrics output
func metricsHandler(tc *tile38.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Query for the extender server statistics and parse out the stats
		out, err := tc.Do("SERVER", "ext")
		if err != nil {
			return err
		}
		m := gjson.Get(out, "stats").Map()

		// Produce a fully populated prometheus metrics output
		var promOutput string
		for _, metric := range metrics {
			promOutput += metric.promString(get(m, metric.Key))
		}

		// Return a fully populated prometheus document
		return c.String(http.StatusOK, promOutput)
	}
}

// metric is a type of struct used to store a metrics type, key and description
type metric struct{ Type, Key, Desc string }

// promString returns the prometheus string representation of the metric given
// a float64 value
func (m metric) promString(val float64) string {
	return fmt.Sprintf("# HELP %s %s\n", m.Key, m.Desc) +
		fmt.Sprintf("# TYPE %s %s\n", m.Key, m.Type) +
		fmt.Sprintf("%s %v\n", m.Key, val)
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
