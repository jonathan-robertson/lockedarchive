package stream_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

func TestFastChunk(t *testing.T) {
	src := []byte("This is a test set of data and it is very nice")
	r := bytes.NewReader(src)
	dst := runChunkTest(t, r)
	verifyBytesEqual(t, src, dst)
}

func TestSlowChunk(t *testing.T) {
	src := []byte("This is a test set of data and it is very nice")
	r := bytes.NewReader(src)
	pr, pw := io.Pipe()
	go func() {
		smallBlock := make([]byte, 2)
		for ; ; time.Sleep(200 * time.Millisecond) { // 200ms to simulate high latency connection
			n, err := r.Read(smallBlock)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}

			if n > 0 {
				if _, writeErr := pw.Write(smallBlock[:n]); writeErr != nil {
					t.Fatal(writeErr)
				}
			}

			if err == io.EOF {
				break
			}
		}
		if err := pw.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	dst := runChunkTest(t, pr)
	verifyBytesEqual(t, src, dst)
}

func runChunkTest(t *testing.T, r io.Reader) []byte {
	var dst []byte

	chunk := make([]byte, 5)
	for {
		length, chunkErr := stream.GetChunk(context.TODO(), r, chunk)
		t.Logf("GetChunk returned length:%d, chunkErr:%v", length, chunkErr)
		if chunkErr != nil {
			if chunkErr != io.EOF {
				t.Fatal(chunkErr)
			} else if length == 0 {
				break
			}
		}

		t.Logf("Reading chunk: %s", chunk[:length])
		dst = append(dst, chunk[:length]...)

		if chunkErr == io.EOF {
			break
		}
	}

	return dst
}

func verifyBytesEqual(t *testing.T, src, dst []byte) {
	if len(src) != len(dst) {
		t.Fatalf("length of data pre and post chunked does not match - before: %d after: %d", len(src), len(dst))
	}

	if !bytes.Equal(src, dst) {
		t.Fatalf("contents of original data and combined chunks does not match\n%s\n%s", src, dst)
	}

	t.Log("successfully chunked and reconstructed data")
}
