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
	Port    string
	Monitor *monitor.Monitor
	listener net.Listener
	mu      sync.Mutex
	closed  bool
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

	// SOCKS5 El Sıkışma (Handshake)
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	n, err := clientConn.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}

	// Kimlik doğrulama gerektirme (No authentication required: 0x00)
	_, err = clientConn.Write([]byte{0x05, 0x00})
	if err != nil {
		return
	}

	// İstemci isteği (CONNECT komutu)
	n, err = clientConn.Read(buf)
	if err != nil || n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
		return
	}

	// Hedef adres tipi ve adresi çözme
	addrType := buf[3]
	var targetAddr string
	var destPort int
	dataIdx := 4

	switch addrType {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		ip := net.IP(buf[dataIdx : dataIdx+4])
		dataIdx += 4
		destPort = int(buf[dataIdx])<<8 | int(buf[dataIdx+1])
		targetAddr = fmt.Sprintf("%s:%d", ip.String(), destPort)
	case 0x03: // Domain name
		addrLen := int(buf[dataIdx])
		dataIdx++
		if n < dataIdx+addrLen+2 {
			return
		}
		domain := string(buf[dataIdx : dataIdx+addrLen])
		dataIdx += addrLen
		destPort = int(buf[dataIdx])<<8 | int(buf[dataIdx+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, destPort)
	case 0x04: // IPv6
		if n < 22 {
			return
		}
		ip := net.IP(buf[dataIdx : dataIdx+16])
		dataIdx += 16
		destPort = int(buf[dataIdx])<<8 | int(buf[dataIdx+1])
		targetAddr = fmt.Sprintf("[%s]:%d", ip.String(), destPort)
	default:
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
