package client

import (
	"bytes"
	"context"
	"github.com/tidwall/resp"
	"io"
	"net"
)

type Client struct {
	addr string
	conn net.Conn
}

func New(addr string) *Client {
	return &Client{
		addr: addr,
	}
}

func (c *Client) Set(ctx context.Context, key string, val string) error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("SET"),
		resp.StringValue(key),
		resp.StringValue(val),
	})
	_, err = io.Copy(conn, buf)
	return err
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	// TODO we dont need to do net.dial everytime. gotta keep it inside CLient if possible
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{
		resp.StringValue("GET"),
		resp.StringValue(key),
	})
	_, err = io.Copy(conn, buf)
	if err != nil {
		return "", err
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	return string(b[:n]), err
}
