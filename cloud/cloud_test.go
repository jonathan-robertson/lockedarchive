package cloud

import (
	"testing"
	"time"

	"github.com/gtank/cryptopasta"
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

	key := cryptopasta.NewEncryptionKey()
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
