package crypt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func init() {
	InitializeBlock("example key 1234")
}

func TestDecrypt(t *testing.T) {
	encryptedInput, err := hex.DecodeString("175ca7c365899c8f03f75e21dce7e2046ad858ea9efee61e1e62325c87cb42ec9440f2e074cc5874a859a7af4033185f")
	if err != nil {
		panic(err)
	}
	expectedOutput := []byte("beepbeepbeepboooooooo-splosion")

	output, err := Decrypt(encryptedInput)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(output, expectedOutput) != 0 {
		t.Fatalf("output does not match expected output\nout:\t%s\nexp:\t%s\n", output, expectedOutput)
	}
}

func TestDecryptHexStringToString(t *testing.T) {
	input := "f363f3ccdcb12bb883abf484ba77d9cd7d32b5baecb3d4b1b3e0e4beffdb3ded"
	expectedOutput := "exampleplaintext"

	output, err := DecryptHexStringToString(input)
	if err != nil {
		t.Fatal(err)
	}

	if output != expectedOutput {
		t.Fatalf("output does not match expected output\noutput: %q\nhex: %x\n\nexpected: %q\nhex: %x\n", output, output, expectedOutput, expectedOutput)
	}
}

func TestEncrypt(t *testing.T) {
	input := []byte{98, 101, 101, 112, 98, 111, 111, 112} // "beepboop"
	expectedOutput := "beepboop"

	output, err := Encrypt(input)
	if err != nil {
		t.Fatal(err)
	}
	hexout := hex.EncodeToString(output)

	decryptedOutput, err := DecryptHexStringToString(hexout)
	if err != nil {
		t.Fatal(err)
	}

	if expectedOutput != decryptedOutput {
		t.Fatalf("output does not match expected output\nout:\t%q\nexp:\t%q\n", decryptedOutput, expectedOutput)
	}
}

func TestEncryptStringToHexString(t *testing.T) {
	input := "000010000"
	expectedOutput := input

	encryptedOutput, err := EncryptStringToHexString(input)
	if err != nil {
		t.Fatal(err)
	}

	decryptedOutput, err := DecryptHexStringToString(encryptedOutput)
	if err != nil {
		t.Fatal(err)
	}

	if decryptedOutput != expectedOutput {
		t.Fatalf("output does not match expected output\nout:\t%q\nexp:\t%q\n", decryptedOutput, expectedOutput)
	}
}
