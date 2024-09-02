package p2p

import (
	"encoding/gob"
	"io"
)

// Decoder is an interface for decoding messages from an io.Reader into an RPC struct.
type Decoder interface {
	Decode(io.Reader, *RPC) error // Decode reads from the provided io.Reader and decodes the data into the given RPC struct.
}

// GOBDecoder is a struct that implements the Decoder interface using Go's gob encoding.
type GOBDecoder struct{}

// Decode reads from the given io.Reader and decodes the data into the provided RPC struct using gob encoding.
func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

// DefaultDecoder is a struct that implements the Decoder interface using custom logic.
type DefaultDecoder struct{}

// Decode reads from the given io.Reader and decodes the data into the provided RPC struct.
// It handles both regular messages and incoming streams.
func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	// Peek at the first byte to determine if the incoming data is a stream.
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil // If there's an error reading the first byte, return nil.
	}

	// Check if the first byte indicates an incoming stream.
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true // Mark the RPC message as a stream.
		return nil        // No further decoding needed for streams.
	}

	// If not a stream, read the remaining data into a buffer.
	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err // Return any error encountered while reading the data.
	}

	// Set the RPC's payload to the data read from the buffer.
	msg.Payload = buf[:n]

	return nil
}
