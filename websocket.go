package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

const wsWriteTimeout = 1200 * time.Millisecond

type simpleWebSocket struct {
	conn      net.Conn
	r         *bufio.Reader
	mu        sync.Mutex
	done      chan struct{}
	doneOnce  sync.Once
	closeOnce sync.Once
}

func dialWebSocket(rawURL string, timeout time.Duration) (*simpleWebSocket, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ws" {
		return nil, fmt.Errorf("unsupported WebSocket scheme %q", u.Scheme)
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		_ = conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(nonce)
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, u.Host, key)
	if _, err := io.WriteString(conn, req); err != nil {
		_ = conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !strings.Contains(status, " 101 ") {
		_ = conn.Close()
		return nil, fmt.Errorf("WebSocket handshake failed: %s", strings.TrimSpace(status))
	}
	headers := map[string]string{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if idx := strings.IndexByte(line, ':'); idx > 0 {
			headers[strings.ToLower(strings.TrimSpace(line[:idx]))] = strings.TrimSpace(line[idx+1:])
		}
	}
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	wantAccept := base64.StdEncoding.EncodeToString(h[:])
	if !strings.EqualFold(headers["sec-websocket-accept"], wantAccept) {
		_ = conn.Close()
		return nil, errors.New("WebSocket server returned an invalid Sec-WebSocket-Accept header")
	}
	_ = conn.SetDeadline(time.Time{})
	ws := &simpleWebSocket{conn: conn, r: br, done: make(chan struct{})}
	go ws.readLoop()
	// No application-level ping loop here. MobiFlight only needs display frames;
	// timed writes plus the reader detect failed/restarted local connections.
	return ws, nil
}

func (w *simpleWebSocket) closeDone() { w.doneOnce.Do(func() { close(w.done) }) }

// Close is deliberately non-blocking with respect to the writer mutex. net.Conn
// supports concurrent Close and Write; closing the connection interrupts a stuck
// write instead of letting the GUI wait behind it.
func (w *simpleWebSocket) Close() error {
	w.closeDone()
	var closeErr error
	w.closeOnce.Do(func() {
		if w.conn != nil {
			_ = w.conn.SetDeadline(time.Now())
			closeErr = w.conn.Close()
		}
	})
	return closeErr
}

func (w *simpleWebSocket) Alive() bool {
	select {
	case <-w.done:
		return false
	default:
		return true
	}
}

func (w *simpleWebSocket) SendText(text string) error {
	if !w.Alive() {
		return errors.New("WebSocket is closed")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.Alive() {
		return errors.New("WebSocket is closed")
	}
	if err := w.writeFrameLocked(0x1, []byte(text)); err != nil {
		w.closeDone()
		_ = w.Close()
		return err
	}
	return nil
}

func (w *simpleWebSocket) writeFrameLocked(opcode byte, payload []byte) error {
	if w.conn == nil {
		return errors.New("socket closed")
	}
	var hdr []byte
	n := len(payload)
	switch {
	case n <= 125:
		hdr = []byte{0x80 | opcode, 0x80 | byte(n)}
	case n <= 65535:
		hdr = make([]byte, 4)
		hdr[0] = 0x80 | opcode
		hdr[1] = 0x80 | 126
		binary.BigEndian.PutUint16(hdr[2:], uint16(n))
	default:
		hdr = make([]byte, 10)
		hdr[0] = 0x80 | opcode
		hdr[1] = 0x80 | 127
		binary.BigEndian.PutUint64(hdr[2:], uint64(n))
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	frame := make([]byte, 0, len(hdr)+4+n)
	frame = append(frame, hdr...)
	frame = append(frame, mask...)
	for i, b := range payload {
		frame = append(frame, b^mask[i&3])
	}
	_ = w.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	_, err := w.conn.Write(frame)
	_ = w.conn.SetWriteDeadline(time.Time{})
	return err
}

func (w *simpleWebSocket) readLoop() {
	defer func() {
		w.closeDone()
		_ = w.Close()
	}()
	for {
		b0, err := w.r.ReadByte()
		if err != nil {
			return
		}
		b1, err := w.r.ReadByte()
		if err != nil {
			return
		}
		opcode := b0 & 0x0f
		masked := (b1 & 0x80) != 0
		plen := uint64(b1 & 0x7f)
		if plen == 126 {
			var x [2]byte
			if _, err = io.ReadFull(w.r, x[:]); err != nil {
				return
			}
			plen = uint64(binary.BigEndian.Uint16(x[:]))
		}
		if plen == 127 {
			var x [8]byte
			if _, err = io.ReadFull(w.r, x[:]); err != nil {
				return
			}
			plen = binary.BigEndian.Uint64(x[:])
		}
		if plen > 1<<20 {
			return
		}
		var mask [4]byte
		if masked {
			if _, err = io.ReadFull(w.r, mask[:]); err != nil {
				return
			}
		}
		payload := make([]byte, int(plen))
		if _, err = io.ReadFull(w.r, payload); err != nil {
			return
		}
		if masked {
			for i := range payload {
				payload[i] ^= mask[i&3]
			}
		}
		switch opcode {
		case 0x8: // close
			return
		case 0x9: // ping -> pong
			w.mu.Lock()
			if w.Alive() {
				_ = w.writeFrameLocked(0xA, payload)
			}
			w.mu.Unlock()
		}
	}
}
