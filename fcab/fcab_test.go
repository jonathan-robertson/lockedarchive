package fcab

import (
	"testing"

	"github.com/puddingfactory/filecabinet/clients/s3client"
)

var (
	// Compliance check
	_ Client = (*s3client.Client)(nil)
)

func TestGenNewID(t *testing.T) {
	t.Log(generateKey())
}

// 32: b3de3ec6dacbc703728e456978eb564a625072da81d093e860234f1e2f672a43
// 16: a837a373b922da4b31e5cfbc845db09d
// 08: 95e23823233dc09c
