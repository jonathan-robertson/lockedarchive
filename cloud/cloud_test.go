package cloud

import (
	"testing"
	"time"

	"github.com/jonathan-robertson/lockedarchive/stream"
)

func TestEntry(t *testing.T) {
	entry := Entry{
		Key:          "123456789",
		ParentKey:    "987654321",
		Name:         "Important.doc",
		IsDir:        false,
		Size:         153432,
		LastModified: time.Now(),
	}

	key := stream.GenerateKey()
	meta, err := entry.Meta(key)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("encoded:\n%s\n%x", meta, meta)

	var newEntry Entry
	if err := newEntry.UpdateMeta(meta, key); err != nil {
		t.Fatal(err)
	}
	t.Logf("success: %+v", newEntry)
}
