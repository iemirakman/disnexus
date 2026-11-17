package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"disnexus/internal/monitor"
)

// bufferPool Garbage Collector yükünü azaltmak için bellek havuzu.
var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 32*1024) // 32KB buffer
	},
}

// Server SOCKS5 proxy sunucusunu temsil eder.
type Server struct {
	Port     string
	Monitor  *monitor.Monitor
	listener net.Listener
	mu       sync.Mutex
	closed   bool
}

// Start proxy sunucusunu başlatır.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errors.New("sunucu kapalı")
	}
	s.mu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:"+s.Port)
	if err != nil {
		return fmt.Errorf("port %s başka bir uygulama tarafından kullanılıyor veya yetki yok: %w", s.Port, err)
	}
	s.listener = ln

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				s.mu.Lock()
				isClosed := s.closed
				s.mu.Unlock()
				if isClosed {
					return
				}
				log.Printf("[proxy] bağlantı kabul edilemedi: %v", err)
				continue
			}
			go s.handleClient(conn)
		}
	}()

	log.Printf("[proxy] SOCKS5 sunucu 127.0.0.1:%s adresinde dinliyor", s.Port)
	return nil
}

// Stop proxy sunucusunu durdurur.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) handleClient(clientConn net.Conn) {
	defer clientConn.Close()

	// Zombi bağlantıları önlemek için zaman aşımı
	_ = clientConn.SetDeadline(time.Now().Add(60 * time.Second))

	targetAddr, err := negotiateSOCKS5(clientConn)
	if err != nil {
		return
	}

	// Round-Robin ile aktif IP seçimi
	localIP, ok := s.Monitor.GetNextIP()
	if !ok {
		log.Println("[proxy] Uyarı: 0 Aktif Hat! İstek reddedildi.")
		// Bağlantı hatası yanıtı gönder (0x01: general SOCKS server failure)
		_, _ = clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// Seçilen yerel ağ adaptörünün IP adresi ile dış dünyaya bağlan
	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: localIP},
		Timeout:   10 * time.Second,
	}

	remoteConn, err := dialer.DialContext(context.Background(), "tcp", targetAddr)
	if err != nil {
		// Hedef erişilemedi yanıtı (0x04: Host unreachable)
		_, _ = clientConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	// Başarılı yanıt gönder (0x00: succeeded)
	_, err = clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return
	}

	// İki yönlü yüksek performanslı trafik aktarımı (Bidirectional Relay with sync.Pool)
	errCh := make(chan error, 2)

	go func() {
		buf1 := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf1)
		_, err := copyWithBuffer(remoteConn, clientConn, buf1)
		errCh <- err
	}()

	go func() {
		buf2 := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf2)
		_, err := copyWithBuffer(clientConn, remoteConn, buf2)
		errCh <- err
	}()

	<-errCh
}

func negotiateSOCKS5(conn net.Conn) (string, error) {
	var greeting [2]byte
	if _, err := io.ReadFull(conn, greeting[:]); err != nil {
		return "", err
	}
	if greeting[0] != 0x05 || greeting[1] == 0 {
		return "", errors.New("invalid SOCKS5 greeting")
	}

	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}
	noAuth := false
	for _, method := range methods {
		if method == 0x00 {
			noAuth = true
			break
		}
	}
	if !noAuth {
		_, _ = conn.Write([]byte{0x05, 0xff})
		return "", errors.New("client does not support no-authentication method")
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", err
	}

	var request [4]byte
	if _, err := io.ReadFull(conn, request[:]); err != nil {
		return "", err
	}
	if request[0] != 0x05 || request[1] != 0x01 || request[2] != 0x00 {
		return "", errors.New("unsupported SOCKS5 request")
	}

	var host string
	switch request[3] {
	case 0x01:
		var ip [net.IPv4len]byte
		if _, err := io.ReadFull(conn, ip[:]); err != nil {
			return "", err
		}
		host = net.IP(ip[:]).String()
	case 0x03:
		var size [1]byte
		if _, err := io.ReadFull(conn, size[:]); err != nil {
			return "", err
		}
		domain := make([]byte, int(size[0]))
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)
	case 0x04:
		var ip [net.IPv6len]byte
		if _, err := io.ReadFull(conn, ip[:]); err != nil {
			return "", err
		}
		host = net.IP(ip[:]).String()
	default:
		return "", errors.New("unsupported SOCKS5 address type")
	}

	var port [2]byte
	if _, err := io.ReadFull(conn, port[:]); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, fmt.Sprint(int(port[0])<<8|int(port[1]))), nil
}

func copyWithBuffer(dst net.Conn, src net.Conn, buf []byte) (written int64, err error) {
	for {
		_ = src.SetReadDeadline(time.Now().Add(90 * time.Second))
		_ = dst.SetWriteDeadline(time.Now().Add(90 * time.Second))

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
