package p2p

import "net"

// Message holds any arbitrary data that is beingg sent
//
//	over eeach transport between two nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
}
