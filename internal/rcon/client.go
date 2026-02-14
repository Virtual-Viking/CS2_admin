package rcon

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"cs2admin/internal/pkg/logger"
)

const rconTimeout = 5 * time.Second

// Client is a single-connection RCON client.
type Client struct {
	addr      string
	password  string
	conn      net.Conn
	mu        sync.Mutex
	requestID int32
	connected bool
}

// NewClient creates a new RCON client. Call Connect to establish connection.
func NewClient(addr, password string) *Client {
	return &Client{
		addr:      addr,
		password:  password,
		requestID: 1,
	}
}

// Connect establishes a TCP connection and authenticates.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	conn, err := net.DialTimeout("tcp", c.addr, rconTimeout)
	if err != nil {
		logger.Log.Debug().Err(err).Str("addr", c.addr).Msg("rcon: dial failed")
		return fmt.Errorf("rcon: connect: %w", err)
	}

	c.conn = conn
	c.connected = true

	if err := c.authenticate(); err != nil {
		c.conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}

	logger.Log.Info().Str("addr", c.addr).Msg("rcon: connected and authenticated")
	return nil
}

// authenticate sends the auth packet and verifies the response.
func (c *Client) authenticate() error {
	reqID := c.nextRequestID()
	p := &Packet{
		RequestID: reqID,
		Type:      PacketTypeAuth,
		Body:      c.password,
	}

	if err := c.sendPacketLocked(p); err != nil {
		return fmt.Errorf("rcon: auth send: %w", err)
	}

	resp, err := c.readPacketLocked()
	if err != nil {
		return fmt.Errorf("rcon: auth read: %w", err)
	}

	// Auth success: server echoes request ID. Auth failure: server sends RequestID -1.
	if resp.RequestID == -1 {
		return fmt.Errorf("rcon: authentication failed")
	}

	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.connected = false
	if err != nil {
		logger.Log.Debug().Err(err).Str("addr", c.addr).Msg("rcon: close failed")
		return err
	}
	logger.Log.Info().Str("addr", c.addr).Msg("rcon: disconnected")
	return nil
}

// Execute sends a command and returns the response.
// Handles multi-packet responses by sending an empty SERVERDATA_RESPONSE_VALUE after the command
// and reading until that sentinel is echoed back.
func (c *Client) Execute(command string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return "", fmt.Errorf("rcon: not connected")
	}

	reqID := c.nextRequestID()

	// Send command
	cmdPkt := &Packet{
		RequestID: reqID,
		Type:      PacketTypeExecCommand,
		Body:      command,
	}
	if err := c.sendPacketLocked(cmdPkt); err != nil {
		return "", fmt.Errorf("rcon: execute send: %w", err)
	}

	// Send empty SERVERDATA_RESPONSE_VALUE as sentinel so server echoes it when done
	sentinelID := c.nextRequestID()
	sentinelPkt := &Packet{
		RequestID: sentinelID,
		Type:      PacketTypeResponseValue,
		Body:      "",
	}
	if err := c.sendPacketLocked(sentinelPkt); err != nil {
		return "", fmt.Errorf("rcon: execute sentinel send: %w", err)
	}

	var out string
	for {
		resp, err := c.readPacketLocked()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("rcon: execute read: %w", err)
		}

		// Empty sentinel echoed back -> we've read all response data
		if resp.RequestID == sentinelID && resp.Type == PacketTypeResponseValue && resp.Body == "" {
			break
		}

		// Collect command response (type 0, matching our command request ID)
		if resp.RequestID == reqID && resp.Type == PacketTypeResponseValue {
			out += resp.Body
		}
	}

	return out, nil
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *Client) nextRequestID() int32 {
	id := c.requestID
	c.requestID++
	if c.requestID < 1 {
		c.requestID = 1
	}
	return id
}

func (c *Client) sendPacketLocked(p *Packet) error {
	data, err := EncodePacket(p)
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(time.Now().Add(rconTimeout))
	n, err := c.conn.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("rcon: incomplete write: %d/%d", n, len(data))
	}
	return nil
}

func (c *Client) readPacketLocked() (*Packet, error) {
	c.conn.SetReadDeadline(time.Now().Add(rconTimeout))
	return ReadPacket(c.conn)
}
