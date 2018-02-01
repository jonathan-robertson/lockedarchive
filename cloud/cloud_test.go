package cloud_test

import (
	"testing"
	"time"

	"github.com/jonathan-robertson/lockedarchive/cloud"
	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	pass = "fishies forever"
)

func TestEntry(t *testing.T) {
	secure.Reset()

	pc, err := secure.ProtectPassphrase([]byte(pass))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Destroy()

	kc, err := secure.GenerateKeyContainer()
	if err != nil {
		t.Fatal(err)
	}
	defer kc.Destroy()

	keyStr, err := secure.EncryptWithSaltToString(pc, kc.Buffer())
	if err != nil {
		t.Fatal(err)
	}

	entry := cloud.Entry{
		ID:           "testID",
		Key:          keyStr,
		ParentID:     "987654321",
		Name:         "Important.doc",
		IsDir:        false,
		Size:         153432,
		LastModified: time.Now().Round(0),
		Mode:         0655,
	}

	meta, err := entry.Meta(pc)
	if err != nil {
		t.Fatal(err)
	}

	otherEntry := cloud.Entry{ID: "test 2"}
	if err := otherEntry.UpdateMeta(pc, meta); err != nil {
		t.Fatal(err)
	}
	assertEquals(t, entry.Key, otherEntry.Key)
	assertEquals(t, entry.ParentID, otherEntry.ParentID)
	assertEquals(t, entry.Name, otherEntry.Name)
	assertEquals(t, entry.IsDir, otherEntry.IsDir)
	assertEquals(t, entry.Size, otherEntry.Size)
	assertEquals(t, entry.LastModified, otherEntry.LastModified)
	assertEquals(t, entry.Mode, otherEntry.Mode)

	t.Log("entry metadata encrypted and decrypted successfully")
}

func assertEquals(t *testing.T, x, y interface{}) {
	if x != y {
		t.Fatalf("%+v != %+v", x, y)
	}
}
