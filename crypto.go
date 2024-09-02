package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// generateID generates a random 32-byte ID and returns it as a hexadecimal string.
func generateID() string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

// hashKey hashes a string key using MD5 and returns the hash as a hexadecimal string.
func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// newEncryptionKey generates a new random 32-byte encryption key.
func newEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

// copyStream reads from the src Reader, applies the cipher stream transformation, and writes to the dst Writer.
// It returns the number of bytes written or an error.
func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024) // Buffer size of 32KB.
		nw  = blockSize             // Initialize nw to block size.
	)
	for {
		n, err := src.Read(buf) // Read from src into the buffer.
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n]) // Apply the XOR transformation on the buffer.
			nn, err := dst.Write(buf[:n])     // Write the transformed data to dst.
			if err != nil {
				return 0, err
			}
			nw += nn // Increment nw by the number of bytes written.
		}
		if err == io.EOF { // Stop reading at the end of the file.
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return nw, nil
}

// copyDecrypt decrypts data from the src Reader and writes the plaintext to the dst Writer.
// It returns the number of bytes written or an error.
func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key) // Create a new AES cipher block using the key.
	if err != nil {
		return 0, err
	}

	// Read the IV from the src Reader. The IV size is equal to the block size.
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)                     // Create a new CTR stream cipher using the block and IV.
	return copyStream(stream, block.BlockSize(), src, dst) // Decrypt and copy the data.
}

// copyEncrypt encrypts data from the src Reader and writes the ciphertext to the dst Writer.
// It returns the number of bytes written or an error.
func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key) // Create a new AES cipher block using the key.
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // Create a random IV with the block size.
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Prepend the IV to the output before the ciphertext.
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)                     // Create a new CTR stream cipher using the block and IV.
	return copyStream(stream, block.BlockSize(), src, dst) // Encrypt and copy the data.
}
