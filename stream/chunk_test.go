package stream_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

func TestChunk(t *testing.T) {
	src := []byte("This is a test set of data and it is very nice")
	r := bytes.NewReader(src)

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

	fmt.Printf("dst: %s", dst)

	if len(src) != len(dst) {
		t.Fatalf("length of data pre and post chunked does not match - before: %d after: %d", len(src), len(dst))
	}

	if !bytes.Equal(src, dst) {
		t.Fatalf("contents of original data and combined chunks does not match\n%s\n%s", src, dst)
	}

	t.Log("successfully chunked and reconstructed data")
}
