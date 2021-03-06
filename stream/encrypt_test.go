package stream_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/secure"
	"github.com/jonathan-robertson/lockedarchive/stream"
)

const (
	encSrcFilename = "src.txt"
	encWrkFilename = "encrypted.txt"
	encDstFilename = "decrypted.txt"
)

func TestEncryption(t *testing.T) {
	kc, err := secure.GenerateKeyContainer()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Encrypt", func(t *testing.T) { runEncryption(t, kc.Key()) })
	t.Run("Decrypt", func(t *testing.T) { runDecryption(t, kc.Key()) })
	compareAndCleanup(t, encSrcFilename, encWrkFilename, encDstFilename)
}

func runEncryption(t *testing.T, key secure.Key) {
	src, dst := setup(t, encSrcFilename, encWrkFilename)
	defer src.Close()
	defer dst.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if stream.IsTooLargeToChunk(srcInfo.Size()) {
		t.Fatal(stream.ErrEncryptSize)
	}

	written, err := stream.Encrypt(context.Background(), key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	// Get stats
	srcStat, err := src.Stat()
	if err != nil {
		t.Fatal(err)
	}
	dstStat, err := dst.Stat()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("srcSize of %d, dstSize of %d with a difference of %d bytes\n", srcStat.Size(), dstStat.Size(), dstStat.Size()-srcStat.Size())

	t.Logf("successfully wrote %d bytes of encrypted data from %s to %s", written, encSrcFilename, encWrkFilename)
}

func runDecryption(t *testing.T, key secure.Key) {
	src, dst := setup(t, encWrkFilename, encDstFilename)
	defer src.Close()
	defer dst.Close()

	written, err := stream.Decrypt(context.Background(), key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	t.Logf("successfully wrote %d bytes of decrypted data from %s to %s", written, encWrkFilename, encDstFilename)
}
