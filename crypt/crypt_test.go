package crypt

import "testing"

func TestEncryptionAndDecryption(t *testing.T) {
	if err := InitializeBlock("example key 1234"); err != nil {
		t.Fatal(err)
	}

	unencryptedString := "exampleplaintext"

	t.Logf("Unencrypted string: %s", unencryptedString)

	encryptedHexString, err := EncryptStringToHexString(unencryptedString)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Encrypted hex string: %s", encryptedHexString)

	// ciphertext, _ := hex.DecodeString("f363f3ccdcb12bb883abf484ba77d9cd7d32b5baecb3d4b1b3e0e4beffdb3ded")
	decryptedString, err := DecryptHexStringToString(encryptedHexString)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Decrypted string: %s", decryptedString)

	if unencryptedString != decryptedString {
		t.Fatal("Unencrypted string and Decrypted string do not match.")
	}
}
