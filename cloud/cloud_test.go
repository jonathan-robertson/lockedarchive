package cloud_test

import (
	"testing"
	"time"

	"github.com/jonathan-robertson/lockedarchive/cloud"
)

func TestEntry(t *testing.T) {
	entry := cloud.Entry{
		Key:          "123456789",
		ParentKey:    "987654321",
		Name:         "Important.doc",
		IsDir:        false,
		Size:         153432,
		LastModified: time.Now(),
		Mode:         nil,
	}

	t.Fail("not implemented")
	// TODO: need to finish deciding on how to encrypt/decrypt entry.Key first
}
