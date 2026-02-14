package rcon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Packet type constants for Source RCON protocol.
const (
	PacketTypeAuth          int32 = 3
	PacketTypeExecCommand   int32 = 2
	PacketTypeResponseValue int32 = 0
	PacketTypeAuthResponse  int32 = 2
)

// Packet represents a Source RCON binary packet.
// Size is computed during encode; it is the length of the remainder (RequestID + Type + Body + null + padding).
type Packet struct {
	Size      int32
	RequestID int32
	Type      int32
	Body      string
}

// EncodePacket encodes a packet to bytes (little-endian).
// Size is computed as: 4 (RequestID) + 4 (Type) + len(Body)+1 (null term) + 1 (padding).
func EncodePacket(p *Packet) ([]byte, error) {
	bodyBytes := []byte(p.Body)
	remainderSize := 4 + 4 + len(bodyBytes) + 1 + 1 // RequestID, Type, body+null, padding
	if remainderSize > 4096 {
		return nil, fmt.Errorf("rcon: packet body too large (%d bytes)", len(p.Body))
	}

	buf := make([]byte, 0, 4+int32(remainderSize))
	enc := make([]byte, 4)

	binary.LittleEndian.PutUint32(enc, uint32(remainderSize))
	buf = append(buf, enc...)

	binary.LittleEndian.PutUint32(enc, uint32(p.RequestID))
	buf = append(buf, enc...)

	binary.LittleEndian.PutUint32(enc, uint32(p.Type))
	buf = append(buf, enc...)

	buf = append(buf, bodyBytes...)
	buf = append(buf, 0, 0) // null terminator + padding

	return buf, nil
}

// DecodePacket decodes bytes to a packet.
// data must include the 4-byte size field and the full remainder.
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("rcon: packet too short (%d bytes)", len(data))
	}

	p := &Packet{}
	p.Size = int32(binary.LittleEndian.Uint32(data[0:4]))
	p.RequestID = int32(binary.LittleEndian.Uint32(data[4:8]))
	p.Type = int32(binary.LittleEndian.Uint32(data[8:12]))

	if len(data) < 13 {
		return nil, fmt.Errorf("rcon: packet too short for body and padding")
	}

	// Body is null-terminated; remainder is body + null + 1 padding byte
	bodyEnd := bytes.IndexByte(data[12:], 0)
	if bodyEnd == -1 {
		return nil, fmt.Errorf("rcon: packet body not null-terminated")
	}
	p.Body = string(data[12 : 12+bodyEnd])

	return p, nil
}

// ReadPacket reads one complete packet from r.
// It reads the 4-byte size first, then the remainder.
func ReadPacket(r io.Reader) (*Packet, error) {
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, sizeBuf); err != nil {
		return nil, err
	}

	size := int32(binary.LittleEndian.Uint32(sizeBuf))
	if size < 0 || size > 4096 {
		return nil, fmt.Errorf("rcon: invalid packet size %d", size)
	}

	remainder := make([]byte, size)
	if _, err := io.ReadFull(r, remainder); err != nil {
		return nil, err
	}

	// Prepend size so DecodePacket gets full packet
	full := append(sizeBuf, remainder...)
	return DecodePacket(full)
}
