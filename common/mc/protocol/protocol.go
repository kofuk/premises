package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type reader struct {
	r   io.Reader
	buf []byte
	pos int
}

type Handler struct {
	conn io.ReadWriter
	r    *reader
}

func (r *reader) Read(buf []byte) (int, error) {
	if len(r.buf)-r.pos < len(buf) {
		rdbuf := make([]byte, 512)
		n, err := r.r.Read(rdbuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				copy(buf, r.buf[r.pos:])
				r.pos += len(r.buf[r.pos:])
				return len(r.buf[r.pos:]), err
			}
			return 0, err
		}
		r.buf = append(r.buf, rdbuf[:n]...)
	}

	copy(buf, r.buf[r.pos:])
	r.pos += len(buf)

	return len(buf), nil
}

type ProtocolHdr struct {
	Length   int
	PacketID int
}

type Handshake struct {
	ProtocolHdr
	Version    int
	ServerAddr string
	ServerPort int
	NextState  int
}

func readVarInt(r io.Reader) (int, error) {
	result := 0
	pos := 0

	for {
		buf := make([]byte, 1)
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		b := buf[0]

		result |= int(b&0x7F) << pos
		if (b & 0x80) == 0 {
			break
		}
		pos += 7

		if pos >= 32 {
			return 0, errors.New("Too long VarInt")
		}
	}

	return result, nil
}

func writeVarInt(w io.Writer, v int) error {
	var bw bytes.Buffer
	for {
		if (v & ^0x7F) == 0 {
			if err := bw.WriteByte(byte(v)); err != nil {
				return err
			}
			break
		}
		if err := bw.WriteByte(byte((v & 0x7F) | 0x80)); err != nil {
			return err
		}
		v >>= 7
	}

	if _, err := w.Write(bw.Bytes()); err != nil {
		return err
	}
	return nil
}

func readShort(r io.Reader) (int, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint16(buf)), nil
}

func readLong(r io.Reader) (int, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint64(buf)), nil
}

func writeLong(w io.Writer, v int) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}

func readPacket(r io.Reader) (*ProtocolHdr, error) {
	result := &ProtocolHdr{}
	var err error

	result.Length, err = readVarInt(r)
	if err != nil {
		return nil, err
	}

	result.PacketID, err = readVarInt(r)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *Handler) ReadHandshake() (*Handshake, error) {
	hdr, err := readPacket(h.r)
	if err != nil {
		return nil, err
	}

	result := &Handshake{
		ProtocolHdr: *hdr,
	}

	result.Version, err = readVarInt(h.r)
	if err != nil {
		return nil, err
	}

	addrLen, err := readVarInt(h.r)
	if err != nil {
		return nil, err
	}
	if addrLen > 255 {
		return nil, errors.New("Address is too long")
	}

	addrBuf := make([]byte, addrLen)
	if _, err := h.r.Read(addrBuf); err != nil {
		return nil, err
	}
	result.ServerAddr = string(addrBuf)

	result.ServerPort, err = readShort(h.r)
	if err != nil {
		return nil, err
	}

	result.NextState, err = readVarInt(h.r)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type Status struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample,omitempty"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Favicon            *string `json:"favicon,omitempty"`
	EnforcesSecureChat bool    `json:"enforcesSecureChat"`
	PreviewsChat       *bool   `json:"prviewsChat"`
	CustomData         any     `json:"x-premises,omitempty"`
}

func (h *Handler) WriteStatus(status Status) error {
	bw := bytes.NewBuffer(nil)
	writeVarInt(bw, 0)

	d, err := json.Marshal(&status)
	if err != nil {
		return err
	}
	writeVarInt(bw, len(d))
	bw.Write(d)

	if err := writeVarInt(h.conn, bw.Len()); err != nil {
		return err
	}
	if _, err := h.conn.Write(bw.Bytes()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ReadStatusRequest() error {
	_, err := readPacket(h.r)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) HandlePingPong() error {
	// Read ping
	hdr, err := readPacket(h.r)
	if err != nil {
		return err
	}

	payload, err := readLong(h.r)
	if err != nil {
		return err
	}
	if hdr.PacketID != 1 {
		return fmt.Errorf("Invalid ping packet: %d", hdr.PacketID)
	}

	// Write pong
	w := bufio.NewWriter(h.conn)
	w.WriteByte(9)
	w.WriteByte(1)
	writeLong(w, payload)
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func NewHandler(conn io.ReadWriter) *Handler {
	return &Handler{
		conn: conn,
		r:    &reader{r: conn},
	}
}

func (h *Handler) OrigBytes() []byte {
	return h.r.buf
}
