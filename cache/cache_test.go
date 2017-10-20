package cache

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

const (
	sourceFilename = "source.txt"
	destFilename   = "dest.txt"
)

// func setup(t *testing.T) {
// 	key = GenerateKey()
// }
// func teardown(t *testing.T) {
// 	if err := os.Remove(destFilename); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func Test(t *testing.T) {
// 	setup(t)
// 	defer teardown(t)
// 	t.Run("Write", func(t *testing.T) { WriteIt(t) })
// 	t.Run("Read", func(t *testing.T) { ReadIt(t) })
// }

// func WriteIt(t *testing.T) {
// 	source, err := os.Open(sourceFilename)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	dest, err := os.OpenFile(destFilename, os.O_CREATE|os.O_WRONLY, 0666)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := Write(source, dest); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func ReadIt(t *testing.T) {
// 	plaintext, err := Read(destFilename)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Logf("%s", plaintext)
// }

// func WriteSecret(t *testing.T) {
// 	source, err := os.Open(sourceFilename)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	dest, err := os.OpenFile(destFilename, os.O_CREATE|os.O_WRONLY, 0666)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var (
// 		message []byte
// 		out     []byte
// 	)

// 	nonce := GenerateNonce()
// 	ciphertext := secretbox.Seal(nonce[:], plaintext, nonce, key)

// }

// TestBasicStream is only for testing the basic streaming functionality of golang
func TestBasicStream(t *testing.T) {
	r := bytes.NewReader([]byte("Tastey fishies!!"))

	chunk := make([]byte, 3)

	for {
		len, err := r.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		t.Logf("%d bytes read for %s", len, string(chunk[:len]))
	}
}

func TestEncryption(t *testing.T) {
	var (
		err            error
		ctx, cancelCtx = context.WithCancel(context.Background())
		pr, pw         = io.Pipe()     // used to route output from encrypt into input for decrypt
		src, dst       = setup(t)      // build setup objects
		key            = GenerateKey() // encryption key to use for this test
	)

	go func() {
		if err = DecryptStream(ctx, key, pr, dst); err != nil {
			cancelCtx()
		}
	}()

	if err = EncryptStream(ctx, key, src, pw); err != nil {
		cancelCtx()
	}

	if err != nil {
		t.Fatal(err)
	}

	if err = src.Close(); err != nil {
		t.Fatal(err)
	}
	if err = dst.Close(); err != nil {
		t.Fatal(err)
	}

	confirmDstMatchesSrc(t)

	teardown(t)
}

func TestCompression(t *testing.T) {
	src, err := os.Open(sourceFilename)
	if err != nil {
		t.Fatal(err)
	}

	mahZip, err := os.OpenFile(destFilename+".zip", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(src)
	written, err := CompressStream(r, mahZip)
	if err != nil {
		t.Fatal(err)
	}

	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	if err := mahZip.Close(); err != nil {
		t.Fatal(err)
	}

	t.Logf("%d bytes compressed", written)
}

func TestDecompression(t *testing.T) {
	src, err := os.Open(destFilename + ".zip")
	if err != nil {
		t.Fatal(err)
	}

	dst, err := os.OpenFile(destFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(src)
	written, err := DecompressStream(r, dst)
	if err != nil {
		t.Fatal(err)
	}

	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}

	t.Logf("%d bytes decompressed", written)
}

// setup returns some starting objects; USER RESPONSIBLE FOR CLOSING src and dst
func setup(t *testing.T) (*os.File, *os.File) {
	src, err := os.Open(sourceFilename)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenFile(destFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}

	return src, dst
}

func teardown(t *testing.T) {
	if err := os.Remove(destFilename); err != nil {
		t.Fatal(err)
	}
}

func confirmDstMatchesSrc(t *testing.T) {
	srcData, err := ioutil.ReadFile(sourceFilename)
	if err != nil {
		t.Fatal(err)
	}

	dstData, err := ioutil.ReadFile(destFilename)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(srcData, dstData) {
		t.Fatal("src and dst are not equal, but they should be")
	}
}
