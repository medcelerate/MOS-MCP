package web

import (
	"net"
	"strconv"
	"testing"
)

func mustListen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

func portOf(ln net.Listener) int {
	return ln.Addr().(*net.TCPAddr).Port
}

func itoa(i int) string { return strconv.Itoa(i) }
