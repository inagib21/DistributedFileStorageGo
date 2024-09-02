# Distributed File Storage System in Go

This project implements a robust, scalable distributed file storage system in Go. It enables secure storage, retrieval, and distribution of files across multiple nodes in a peer-to-peer (P2P) network. The system utilizes TCP for inter-node communication and incorporates encryption for enhanced file security during storage and transfer.

## Features

- **Peer-to-Peer Network**: Nodes communicate over a TCP-based P2P network.
- **File Encryption**: Files are encrypted before storage and decrypted upon retrieval.
- **Content-Addressable Storage**: Files are stored and retrieved based on their content hash.
- **Concurrent File Operations**: Multiple files can be stored and retrieved concurrently.
- **Custom Message Encoding**: Supports both `gob` and custom encoding for network messages.

## System Architecture

The system consists of multiple interconnected nodes, each capable of storing and retrieving files. Key components include:

- **File Servers**: Handle file storage, retrieval, and peer connections.
- **TCP Transport**: Manages network communication between nodes.
- **Encryption Module**: Ensures secure file storage and transfer.
- **Content-Addressable Storage**: Implements efficient file indexing and retrieval.

## Getting Started

### Prerequisites

- Go 1.20 or later
- Make

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/DistributedFileStorageGo.git
   cd DistributedFileStorageGo
   ```

2. Build the project:

   ```bash
   make build
   ```

3. Run the application:

   ```bash
   make run 
   ```

### Configuration

The system uses default configuration values defined in the code. Key settings include:

- Network ports (defined in `main.go`)
- Encryption key size (defined in `crypto.go`)
- Storage root directory (defined in `store.go`)

## Usage

The application initializes three file servers listening on different ports. Here's a basic usage scenario:

1. **Start the servers**: 
   The `main.go` file initializes three servers:
   - Server 1: Port 3000
   - Server 2: Port 7000
   - Server 3: Port 5000 (connects to 3000 and 7000 as bootstrap nodes)

2. **Store Files**: 
   ```go
   err := server.Store("picture_1.png", fileData)
   if err != nil {
       log.Fatal(err)
   }
   ```

3. **Retrieve Files**: 
   ```go
   data, err := server.Retrieve("picture_1.png")
   if err != nil {
       log.Fatal(err)
   }
   ```



## Testing

Tests validate the functionality of encryption, decryption, and TCP transport. To run the tests:

```bash
make test
```

## Code Overview

### `main.go`

The `main.go` file is the entry point of the application. It sets up the servers, initializes the network, and handles the storing and retrieving of files.

### `crypto.go` & `crypto_test.go`

- **Encryption**: Uses AES in CTR mode for encrypting and decrypting files.
- **Key Management**: Generates random encryption keys and handles the initialization vectors (IVs) necessary for AES encryption.
- **Testing**: Validates the encryption and decryption functionality to ensure data integrity.

### `tcp_transport.go` & `tcp_transport_test.go`

- **TCP Transport**: Manages connections between peers in the P2P network, handling both inbound and outbound connections.
- **Message Handling**: Includes functions for sending and receiving data over TCP connections.
- **Testing**: Ensures that the TCP transport functions as expected, handling connections and data transfer correctly.

### `encoding.go`

- **Message Encoding/Decoding**: Provides two implementations (`GOBDecoder` and `DefaultDecoder`) for decoding messages received over the network.
- **Stream Handling**: The `DefaultDecoder` can distinguish between regular messages and incoming streams.

### `store.go`

- **File Storage**: Implements the local file storage system using a content-addressable approach.
- **Path Transformation**: Provides functions to transform file keys into storage paths.

### `server.go`

- **File Server**: Implements the core functionality for storing and retrieving files across the network.
- **Peer Management**: Handles connections with other nodes in the network.

## Performance and Scalability

The system is designed to handle concurrent operations efficiently. Performance metrics and scalability information will be added as the project evolves.

## Error Handling and Troubleshooting

Common errors and their solutions:

- Connection refused: Ensure all servers are running and ports are open.
- File not found: Check if the file exists and the key is correct.


## Future Improvements / Roadmap

- Implement data replication for improved reliability
- Add support for file versioning
- Develop a web-based user interface for easier file management
