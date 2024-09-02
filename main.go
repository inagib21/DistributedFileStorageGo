package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/inagib21/DistributedFileStorageGo/p2p"
)

// makeServer initializes and returns a new FileServer instance with a TCP transport.
// It sets up the server with encryption, storage, and peer management.
func makeServer(listenAddr string, nodes ...string) *FileServer {
	// Define TCP transport options, including the listening address and handshake function.
	tcptransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,           // Address on which the server listens for connections.
		HandshakeFunc: p2p.NOPHandshakeFunc, // No-operation handshake function (does nothing).
		Decoder:       p2p.DefaultDecoder{}, // Default message decoder for incoming data.
	}
	// Create a new TCP transport instance based on the options provided.
	tcpTransport := p2p.NewTCPTransport(tcptransportOpts)

	// Define options for the FileServer, including encryption, storage path, and peer nodes.
	fileServerOpts := FileServerOpts{
		EncKey:            newEncryptionKey(),      // Encryption key for securing data.
		StorageRoot:       listenAddr + "_network", // Root directory for file storage based on the listening address.
		PathTransformFunc: CASPathTransformFunc,    // Function to transform file paths into content-addressable paths.
		Transport:         tcpTransport,            // Set the transport mechanism to the TCP transport created earlier.
		BootstrapNodes:    nodes,                   // List of initial nodes to connect with for bootstrapping the network.
	}

	// Create a new FileServer instance using the options defined above.
	s := NewFileServer(fileServerOpts)

	// Set the OnPeer callback function for handling new peer connections.
	tcpTransport.OnPeer = s.OnPeer

	return s
}

func (fs *FileServer) Start() error {
	return fs.Transport.ListenAndAccept()
}

func main() {
	// Create three FileServer instances listening on different ports.
	// s1 listens on port 3000 with no bootstrap nodes.
	s1 := makeServer(":3000", "")
	// s2 listens on port 7000 with no bootstrap nodes.
	s2 := makeServer(":7000", "")
	// s3 listens on port 5000 and boots with nodes at ports 3000 and 7000.
	s3 := makeServer(":5000", ":3000", ":7000")

	// Start s1 in a separate goroutine and log any fatal errors.
	go func() { log.Fatal(s1.Start()) }()
	// Pause to ensure s1 has started before proceeding.
	time.Sleep(500 * time.Millisecond)

	// Start s2 in a separate goroutine and log any fatal errors.
	go func() { log.Fatal(s2.Start()) }()

	// Pause to ensure both s1 and s2 are running before starting s3.
	time.Sleep(2 * time.Second)

	// Start s3 in a separate goroutine without logging errors to avoid blocking.
	go s3.Start()
	// Pause to allow s3 to connect to s1 and s2.
	time.Sleep(2 * time.Second)

	// Store and retrieve files in a loop to test the file server functionality.
	for i := 0; i < 20; i++ {
		// Generate a key for each file (e.g., "picture_1.png").
		key := fmt.Sprintf("picture_%d.png", i)
		// Create a reader for the file data.
		data := bytes.NewReader([]byte("my big data file here!"))

		// Store the file on the s3 server.
		s3.Store(key, data)

		// Delete the file from local storage on s3 to simulate fetching from the network.
		if err := s3.store.Delete(s3.ID, key); err != nil {
			log.Fatal(err)
		}

		// Attempt to retrieve the file using the s3 server.
		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		// Read the retrieved file data into a byte slice.
		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		// Print the contents of the retrieved file to verify correctness.
		fmt.Println(string(b))
	}
}
