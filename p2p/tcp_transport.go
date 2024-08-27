package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	conn     net.Conn // Underlying connection of the peer
	outbound bool     // Outbound flag, true if this peer was dialed out
}

// NewTCPPeer creates a new TCPeer instance
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// close implemens the peer interface
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	// Address to listen on
	Listener net.Listener
	rpcch    chan RPC

	// Underlying listener
	mu    sync.RWMutex          // Mutex for synchronization
	peers map[net.Addr]*TCPPeer // Map of peers by address
}

// NewTCPTransport creates a new TCPTransport instance
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// Consume implements the Transpoort interfacem which will returnn
// read only channel for reading the incoming messages recieved
// from anoher peer in  the network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// ListenAndAccept listens on the TCP address and starts accepting connections
func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.Listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return nil
}

// startAcceptLoop continuously accepts incoming connections
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.Listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
			continue
		}

		fmt.Printf("New incoming connection from %v\n", conn)

		go t.handleConn(conn)

	}
}

type Temp struct{}

// handleConn handles the incoming connection
func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		fmt.Printf("dropping peer connection: %s ", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, true)

	if err := t.HandshakeFunc(peer); err != nil {

		return
	}

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}
	//Read Loop
	rpc := RPC{}

	for {

		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			fmt.Printf("TCP error: %s\n ", err)
			continue
		}

		rpc.From = conn.RemoteAddr()

		t.rpcch <- rpc
	}

}
