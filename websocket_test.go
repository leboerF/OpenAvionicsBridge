package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestWebSocketHandshakeAndTextFrame(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		var key string
		for {
			line, er := r.ReadString('\n')
			if er != nil {
				return
			}
			if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
				key = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			}
			if line == "\r\n" {
				break
			}
		}
		h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		accept := base64.StdEncoding.EncodeToString(h[:])
		io.WriteString(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "+accept+"\r\n\r\n")
		b0, _ := r.ReadByte()
		_ = b0
		b1, _ := r.ReadByte()
		n := int(b1 & 0x7f)
		if n == 126 {
			var x [2]byte
			io.ReadFull(r, x[:])
			n = int(binary.BigEndian.Uint16(x[:]))
		}
		var mask [4]byte
		io.ReadFull(r, mask[:])
		p := make([]byte, n)
		io.ReadFull(r, p)
		for i := range p {
			p[i] ^= mask[i&3]
		}
		got <- string(p)
		time.Sleep(100 * time.Millisecond)
	}()
	ws, err := dialWebSocket("ws://"+ln.Addr().String()+"/test", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	if err := ws.SendText("hello"); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-got:
		if s != "hello" {
			t.Fatalf("got %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
