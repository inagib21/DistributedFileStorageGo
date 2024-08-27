package p2p

// Perr is an interface that represents the remote node.
type Peer interface {
	Close() error
}

// Transport is anything that handles the communicaation
// between the nodes in the network. This can be of the
// form (TCP, UDP, websockets, ... )
type Transport interface {
	ListenAndAccept() error
	consume() <-chan RPC
}
