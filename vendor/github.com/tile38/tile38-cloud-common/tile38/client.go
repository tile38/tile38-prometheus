package tile38

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/sjson"
)

// Client contains the pool of Tile38 connections
type Client struct{ Pool *redis.Pool }

// New creates a new Tile38 Client that contains a pool of redis connections
func New(addr string) *Client {
	return &Client{Pool: &redis.Pool{
		MaxIdle:     16,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			c.Send("OUTPUT", "json")
			return c, nil
		}, TestOnBorrow: func(conn redis.Conn, _ time.Time) error {
			if resp, _ := redis.String(conn.Do("PING")); resp != "PONG" {
				return errors.New("expected PONG")
			}
			return nil
		},
	}}
}

// Close closes the pool of connections to Tile38
func (c *Client) Close() {
	defer c.Pool.Close()
}

// Do retrieves a new connection from the Tile38 connection pool, sends the
// passed command and closes the connection
func (c *Client) Do(cmd string, args ...interface{}) (string, error) {
	conn := c.Pool.Get()
	defer conn.Close()

	// Perform the request to tile38
	res, err := redis.String(conn.Do(cmd, args...))
	if err != nil {
		return "", err
	}

	// Strip out the elapsed statistic and return the response
	return sjson.Delete(res, "elapsed")
}
