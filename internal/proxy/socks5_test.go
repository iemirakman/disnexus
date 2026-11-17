package proxy

import (
	"io"
	"net"
	"testing"
)

func TestNegotiateSOCKS5ReadsFragmentedFrames(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	type result struct {
		target string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		target, err := negotiateSOCKS5(server)
		done <- result{target: target, err: err}
	}()

	for _, b := range []byte{0x05, 0x01, 0x00} {
		if _, err := client.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	methodReply := make([]byte, 2)
	if _, err := io.ReadFull(client, methodReply); err != nil {
		t.Fatal(err)
	}
	if string(methodReply) != string([]byte{0x05, 0x00}) {
		t.Fatalf("unexpected method reply: %v", methodReply)
	}

	for _, b := range []byte{0x05, 0x01, 0x00, 0x03, 0x0b, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0x01, 0xbb} {
		if _, err := client.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}

	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.target != "example.com:443" {
		t.Fatalf("target = %q, want example.com:443", got.target)
	}
}

func TestNegotiateSOCKS5RejectsUnsupportedAuthentication(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		_, err := negotiateSOCKS5(server)
		done <- err
	}()

	if _, err := client.Write([]byte{0x05, 0x01, 0x02}); err != nil {
		t.Fatal(err)
	}
	methodReply := make([]byte, 2)
	if _, err := io.ReadFull(client, methodReply); err != nil {
		t.Fatal(err)
	}
	if string(methodReply) != string([]byte{0x05, 0xff}) {
		t.Fatalf("unexpected method reply: %v", methodReply)
	}
	if err := <-done; err == nil {
		t.Fatal("expected unsupported authentication method to fail")
	}
}
