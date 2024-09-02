package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/inagib21/DistributedFileStorageGo/p2p"
)

// FileServerOpts holds configuration options for the FileServer
type FileServerOpts struct {
	ID                string            // Unique identifier for the FileServer
	EncKey            []byte            // Encryption key used for file encryption
	StorageRoot       string            // Root directory for file storage
	PathTransformFunc PathTransformFunc // Function to transform file paths
	Transport         p2p.Transport     // Transport layer for peer-to-peer communication
	BootstrapNodes    []string          // List of bootstrap nodes to connect to in the network
}

// FileServer represents a server that handles file storage and retrieval over a network
type FileServer struct {
	FileServerOpts // Embeds FileServerOpts to inherit its fields

	peerLock sync.Mutex          // Mutex to protect concurrent access to peers map
	peers    map[string]p2p.Peer // Map of connected peers identified by their network address

	store  *Store        // Store represents the file storage and management system
	quitch chan struct{} // Channel to signal the server to stop its operation
}

// NewFileServer initializes a new FileServer with the provided options
func NewFileServer(opts FileServerOpts) *FileServer {
	// Configure the storage options for the server
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,       // Set the storage root directory
		PathTransformFunc: opts.PathTransformFunc, // Set the path transformation function
	}

	// Generate a unique ID for the server if not provided
	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}

	// Return a new FileServer instance
	return &FileServer{
		FileServerOpts: opts,                      // Assign the provided options to the server
		store:          NewStore(storeOpts),       // Initialize the file storage system
		quitch:         make(chan struct{}),       // Initialize the quit channel
		peers:          make(map[string]p2p.Peer), // Initialize the peers map
	}
}

// broadcast sends a message to all connected peers
func (s *FileServer) broadcast(msg *Message) error {
	// Encode the message into a byte buffer
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err // Return error if encoding fails
	}

	// Send the encoded message to all peers
	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage}) // Notify peer of incoming message
		if err := peer.Send(buf.Bytes()); err != nil {
			return err // Return error if sending fails
		}
	}

	return nil // Return nil if broadcasting succeeds
}

// Message represents a generic message to be exchanged between peers
type Message struct {
	Payload any // Payload contains the actual data of the message
}

// MessageStoreFile is a specific message type used to store a file
type MessageStoreFile struct {
	ID   string // Unique identifier of the file
	Key  string // Key used to encrypt the file
	Size int64  // Size of the file in bytes
}

// MessageGetFile is a specific message type used to retrieve a file
type MessageGetFile struct {
	ID  string // Unique identifier of the file
	Key string // Key used to identify the file
}

// Get retrieves a file from the local storage or network if not found locally
func (s *FileServer) Get(key string) (io.Reader, error) {
	// Check if the file exists locally
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(s.ID, key) // Read the file from local storage
		return r, err                        // Return the file reader and any error encountered
	}

	// If the file is not found locally, attempt to fetch it from the network
	fmt.Printf("[%s] don't have file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

	// Prepare a message to request the file from peers
	msg := Message{
		Payload: MessageGetFile{
			ID:  s.ID,         // Include the server's ID
			Key: hashKey(key), // Include the hashed key of the file
		},
	}

	// Broadcast the request to all connected peers
	if err := s.broadcast(&msg); err != nil {
		return nil, err // Return error if broadcasting fails
	}

	time.Sleep(time.Millisecond * 500) // Wait for a short duration to receive responses

	// Iterate through peers to receive the file
	for _, peer := range s.peers {
		// Read the file size from the peer connection
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize) // Read file size as int64

		// Write the received file data to local storage
		n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err // Return error if writing fails
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)", s.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream() // Close the peer's data stream
	}

	// Read and return the file from local storage after receiving it from the network
	_, r, err := s.store.Read(s.ID, key)
	return r, err
}

// Store saves a file to local storage and broadcasts it to peers
func (s *FileServer) Store(key string, r io.Reader) error {
	// Create a buffer to hold the file data temporarily
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer) // TeeReader allows reading and copying simultaneously
	)

	// Write the file data to local storage
	size, err := s.store.Write(s.ID, key, tee)
	if err != nil {
		return err // Return error if writing fails
	}

	// Prepare a message to notify peers about the stored file
	msg := Message{
		Payload: MessageStoreFile{
			ID:   s.ID,         // Include the server's ID
			Key:  hashKey(key), // Include the hashed key of the file
			Size: size + 16,    // Include the size of the file
		},
	}

	// Broadcast the stored file information to all connected peers
	if err := s.broadcast(&msg); err != nil {
		return err // Return error if broadcasting fails
	}

	time.Sleep(time.Millisecond * 5) // Wait for a short duration before sending the file

	// Send the file to all connected peers
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer) // Append each peer to the list of writers
	}
	mw := io.MultiWriter(peers...)       // Create a MultiWriter to send the file to multiple peers simultaneously
	mw.Write([]byte{p2p.IncomingStream}) // Notify peers of an incoming file stream
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err // Return error if copying fails
	}

	fmt.Printf("[%s] received and written (%d) bytes to disk\n", s.Transport.Addr(), n)

	return nil // Return nil if the file was stored successfully
}

// Stop gracefully stops the FileServer by closing the quit channel
func (s *FileServer) Stop() {
	close(s.quitch) // Signal the server to stop its operation
}

// OnPeer is triggered when a new peer connects to the server
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()         // Acquire the peer lock to safely modify the peers map
	defer s.peerLock.Unlock() // Ensure the lock is released after the function exits

	s.peers[p.RemoteAddr().String()] = p // Add the new peer to the peers map

	log.Printf("connected with remote %s", p.RemoteAddr()) // Log the new connection

	return nil // Return nil if the peer was successfully added
}

// loop continuously handles incoming messages and peer connections
func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to error or user quit action")
		s.Transport.Close() // Ensure the transport layer is closed when the server stops
	}()

	// Continuously listen for incoming messages or quit signal
	for {
		select {
		case rpc := <-s.Transport.Consume(): // Receive a new RPC (Remote Procedure Call) from the transport layer
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err) // Log decoding errors

			}
		}
	}
}
