package main

import (
	"bytes"
	"fmt"
	"testing"
)

// TestCopyEncryptDecrypt tests the encryption and decryption process using copyEncrypt and copyDecrypt.
func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo not bar"                // The original data to be encrypted.
	src := bytes.NewReader([]byte(payload)) // Source reader for the original data.
	dst := new(bytes.Buffer)                // Destination buffer to hold the encrypted data.
	key := newEncryptionKey()               // Generate a new encryption key.

	// Encrypt the data from src and write it to dst.
	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err) // Report an error if encryption fails.
	}

	fmt.Println(len(payload))      // Print the length of the original data.
	fmt.Println(len(dst.String())) // Print the length of the encrypted data.

	out := new(bytes.Buffer) // Buffer to hold the decrypted data.

	// Decrypt the data from dst and write it to out.
	nw, err := copyDecrypt(key, dst, out)
	if err != nil {
		t.Error(err) // Report an error if decryption fails.
	}

	// Verify that the number of written bytes matches the expected size (IV + payload).
	if nw != 16+len(payload) { // 16 bytes for IV and the length of the payload.
		t.Fail() // Mark the test as failed if the size doesn't match.
	}

	// Verify that the decrypted data matches the original payload.
	if out.String() != payload {
		t.Errorf("decryption failed!!!")
	}
}
