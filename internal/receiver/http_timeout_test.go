package receiver

import (
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
)

// TestHTTPReceiverReadTimeout verifies that the OTLP receiver closes a
// connection that sends headers but never delivers the complete body within
// ReadTimeout.
func TestHTTPReceiverReadTimeout(t *testing.T) {
	r := NewHTTPReceiver(":0", storage.NewStorage(storage.DefaultConfig()))

	// Override the server's ReadTimeout to a short value so the test finishes
	// quickly while still exercising the same code path.
	const shortTimeout = 150 * time.Millisecond
	r.server.ReadTimeout = shortTimeout

	ts := httptest.NewUnstartedServer(r.server.Handler)
	ts.Config.ReadTimeout = shortTimeout
	ts.Start()
	defer ts.Close()

	// Open a raw TCP connection and send partial HTTP headers with no blank
	// line terminator so the server never dispatches the handler — it stays
	// stuck reading headers until ReadTimeout fires and closes the connection.
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Incomplete headers: no final \r\n, so HTTP parsing never completes.
	_, err = conn.Write([]byte(
		"POST /v1/metrics HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Content-Length: 1048576\r\n",
		// intentionally no trailing \r\n
	))
	if err != nil {
		t.Fatalf("write headers: %v", err)
	}

	// The server must close the connection within ReadTimeout + a small margin.
	deadline := time.Now().Add(shortTimeout + 500*time.Millisecond)
	conn.SetReadDeadline(deadline)

	buf := make([]byte, 1)
	_, readErr := conn.Read(buf)
	if readErr == nil {
		t.Fatal("expected connection to be closed by server, but read succeeded")
	}
	if time.Now().After(deadline) {
		t.Fatalf("server did not close the connection within ReadTimeout + margin")
	}
}
