package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

// Client is a Go client for the control protocol (used by tests and the CLI;
// the Flutter app has its own Dart implementation).
type Client struct {
	conn    net.Conn
	nextID  atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan Response
	events  chan RawEvent
	enc     *json.Encoder
	encMu   sync.Mutex
	done    chan struct{}
}

// RawEvent is a pushed event with undecoded data.
type RawEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Dial connects and authenticates.
func Dial(ctx context.Context, addr, token string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:    conn,
		pending: map[int64]chan Response{},
		events:  make(chan RawEvent, 1024),
		enc:     json.NewEncoder(conn),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	if err := c.Call(ctx, "hello", HelloParams{Token: token}, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// Events returns pushed events. Events are dropped if the channel is full.
func (c *Client) Events() <-chan RawEvent { return c.events }

// Close disconnects.
func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) readLoop() {
	defer close(c.done)
	sc := bufio.NewScanner(c.conn)
	sc.Buffer(make([]byte, 64<<10), maxLine)
	for sc.Scan() {
		var probe struct {
			ID    int64  `json:"id"`
			Event string `json:"event"`
		}
		line := append([]byte(nil), sc.Bytes()...)
		if json.Unmarshal(line, &probe) != nil {
			continue
		}
		if probe.Event != "" {
			var ev RawEvent
			if json.Unmarshal(line, &ev) == nil {
				select {
				case c.events <- ev:
				default:
				}
			}
			continue
		}
		var raw struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *Error          `json:"error"`
		}
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		c.mu.Lock()
		ch := c.pending[raw.ID]
		delete(c.pending, raw.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- Response{ID: raw.ID, Result: raw.Result, Error: raw.Error}
		}
	}
}

// Call sends a request and decodes the result into out (which may be nil).
func (c *Client) Call(ctx context.Context, method string, params, out any) error {
	id := c.nextID.Add(1)
	ch := make(chan Response, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = b
	}
	c.encMu.Lock()
	err := c.enc.Encode(Request{ID: id, Method: method, Params: raw})
	c.encMu.Unlock()
	if err != nil {
		return err
	}
	select {
	case resp := <-ch:
		if resp.Error != nil {
			return resp.Error
		}
		if out != nil && resp.Result != nil {
			return json.Unmarshal(resp.Result.(json.RawMessage), out)
		}
		return nil
	case <-c.done:
		return errors.New("connection closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}
