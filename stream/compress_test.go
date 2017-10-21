package stream_test

import (
	"testing"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

const (
	cmpSrcFilename = "src.txt"
	cmpWrkFilename = "compressed.txt"
	cmpDstFilename = "decompressed.txt"
)

func TestCompression(t *testing.T) {
	t.Run("Compress", func(t *testing.T) { runCompression(t) })
	t.Run("Decompress", func(t *testing.T) { runDecompression(t) })
	compareAndCleanup(t, cmpSrcFilename, cmpWrkFilename, cmpDstFilename)
}

func runCompression(t *testing.T) {
	src, dst := setup(t, cmpSrcFilename, cmpWrkFilename)
	defer src.Close()
	defer dst.Close()

	written, err := stream.Compress(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	t.Logf("successfully compressed %d bytes of data from %s to %s", written, cmpSrcFilename, cmpWrkFilename)
}

func runDecompression(t *testing.T) {
	src, dst := setup(t, cmpWrkFilename, cmpDstFilename)
	defer src.Close()
	defer dst.Close()

	written, err := stream.Decompress(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if err := dst.Sync(); err != nil {
		t.Fatal(err)
	}

	t.Logf("successfully compressed %d bytes of data from %s to %s", written, cmpWrkFilename, cmpDstFilename)
}
