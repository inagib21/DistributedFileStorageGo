package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTCPTransport is a basic test for the TCPTransport functionality.
func TestTCPTransport(t *testing.T) {
	// Set up TCPTransport options with a listening address, a no-op handshake function, and a default decoder.
	opts := TCPTransportOpts{
		ListenAddr:    ":3000",          // Address where the TCPTransport should listen for incoming connections.
		HandshakeFunc: NOPHandshakeFunc, // No-op handshake function used for testing.
		Decoder:       DefaultDecoder{}, // Default decoder implementation for testing.
	}

	// Create a new TCPTransport instance using the provided options.
	tr := NewTCPTransport(opts)

	// Assert that the ListenAddr is correctly set.
	assert.Equal(t, tr.ListenAddr, ":3000")

	// Test if the TCPTransport can start listening and accepting connections without errors.
	assert.Nil(t, tr.ListenAndAccept())
}
