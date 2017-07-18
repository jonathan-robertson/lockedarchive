package cache

import (
	"os"
	"testing"

	"github.com/puddingfactory/lockedarchive/cryptopasta"
)

const (
	sourceFilename = "source.txt"
	destFilename   = "dest.txt"
)

func setup(t *testing.T) {
	key = cryptopasta.NewEncryptionKey()
}
func teardown(t *testing.T) {
	if err := os.Remove(destFilename); err != nil {
		t.Fatal(err)
	}
}

func Test(t *testing.T) {
	setup(t)
	defer teardown(t)
	t.Run("Write", func(t *testing.T) { WriteIt(t) })
	t.Run("Read", func(t *testing.T) { ReadIt(t) })
}

func WriteIt(t *testing.T) {
	source, err := os.Open(sourceFilename)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := os.OpenFile(destFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}

	if err := Write(source, dest); err != nil {
		t.Fatal(err)
	}
}

func ReadIt(t *testing.T) {
	plaintext, err := Read(destFilename)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", plaintext)
}
