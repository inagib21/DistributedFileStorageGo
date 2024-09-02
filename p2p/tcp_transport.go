package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer represents a peer in the network connected via a TCP connection.
type TCPPeer struct {
	net.Conn                 // The underlying TCP connection.
	outbound bool            // Indicates whether the connection is outbound or inbound.
	wg       *sync.WaitGroup // WaitGroup to manage stream synchronization.
}

// NewTCPPeer creates and returns a new TCPPeer instance.
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// CloseStream signals that the stream has been closed by decrementing the WaitGroup counter.
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send writes a byte slice to the peer's TCP connection.
func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

// TCPTransportOpts contains configuration options for TCPTransport.
type TCPTransportOpts struct {
	ListenAddr    string           // Address where the transport listens for incoming connections.
	HandshakeFunc HandshakeFunc    // Function for performing the handshake process.
	Decoder       Decoder          // Decoder for decoding incoming messages.
	OnPeer        func(Peer) error // Callback function triggered when a new peer is connected.
}

// TCPTransport manages the TCP connections for a node in the network.
type TCPTransport struct {
	TCPTransportOpts              // Embedding the options struct to inherit its fields.
	listener         net.Listener // Listener for accepting incoming connections.
	rpcch            chan RPC     // Channel for handling incoming RPC messages.
}

// NewTCPTransport creates a new TCPTransport instance with the provided options.
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC, 1024), // Buffered channel for RPCs with a capacity of 1024.
	}
}

// Addr returns the listening address of the transport.
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// Consume returns a read-only channel for consuming incoming RPCs.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// Close closes the TCP listener.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial attempts to establish an outbound TCP connection to the specified address.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true) // Handle the connection in a separate goroutine.

	return nil
}

// ListenAndAccept starts the TCP listener and begins accepting incoming connections.
func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr) // Start listening on the specified address.
	if err != nil {
		return err
	}

	go t.startAcceptLoop() // Start the loop to accept connections in a separate goroutine.

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddr)

	return nil
}

// startAcceptLoop continuously accepts new connections and handles them.
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept() // Accept an incoming connection.
		if errors.Is(err, net.ErrClosed) {
			return // Exit if the listener has been closed.
		}

		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err) // Log any errors that occur during acceptance.
		}

		go t.handleConn(conn, false) // Handle the accepted connection in a separate goroutine.
	}
}

// handleConn handles the TCP connection, performing the handshake and processing incoming RPCs.
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection: %s", err) // Log the reason for dropping the connection.
		conn.Close()                                    // Ensure the connection is closed.
	}()

	peer := NewTCPPeer(conn, outbound) // Create a new TCPPeer for this connection.

	// Perform the handshake using the provided HandshakeFunc.
	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	// If an OnPeer callback is provided, execute it.
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop to process incoming RPCs from the peer.
	for {
		rpc := RPC{}
		// Decode the incoming RPC from the connection.
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}

		rpc.From = conn.RemoteAddr().String() // Set the source address of the RPC.

		// If the RPC is a stream, manage it with the WaitGroup.
		if rpc.Stream {
			peer.wg.Add(1) // Increment the WaitGroup counter to wait for the stream.
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			peer.wg.Wait() // Wait for the stream to be closed.
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}

		// Send the decoded RPC to the transport's RPC channel for further processing.
		t.rpcch <- rpc
	}
}
