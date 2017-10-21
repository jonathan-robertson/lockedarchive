package stream

import (
	"compress/gzip"
	"io"
)

// Compress compresses a stream of data
// TODO: allow other compression methods based on settings
func Compress(r io.Reader, w io.Writer) (int64, error) {
	zw, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(zw, r)
	if err != nil {
		zw.Close()
		return 0, err
	}

	return written, zw.Close()
}

// Decompress decompresses a stream of data
func Decompress(r io.Reader, w io.Writer) (int64, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(w, zr)
	if err != nil {
		zr.Close()
		return 0, err
	}

	return written, zr.Close()
}
