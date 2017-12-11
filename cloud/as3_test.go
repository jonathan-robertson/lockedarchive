package cloud_test

import (
	"testing"

	"github.com/jonathan-robertson/lockedarchive/cloud"
	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	testPassphrase = "passphrase"
)

func TestAS3(t *testing.T) {
	client := setupAS3(t)
	entry := cloud.Entry{
		Key:   "123",
		IsDir: true,
	}

	pc, err := secure.ProtectPassphrase([]byte(testPassphrase))
	if err != nil {
		t.Fatal(err)
	}
	meta, err := entry.Meta(pc)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Upload", func(t *testing.T) {
		if err := client.Upload(entry.ID, meta, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("Head", func(t *testing.T) {
		if err := client.Head(entry); err != nil {
			t.Error(err)
		}
	})
	t.Run("Download", func(t *testing.T) {
		rc, err := client.Download(entry)
		if err != nil {
			t.Error(err)
		}
		rc.Close()
	})
	t.Run("List", func(t *testing.T) {
		entries := make(chan cloud.Entry)
		go func() {
			if err := client.List(entries); err != nil {
				t.Error(err)
			}
		}()
		for entry := range entries {
			t.Logf("list: %+v", entry)
		}
	})
	t.Run("Delete", func(t *testing.T) {
		if err := client.Delete(entry); err != nil {
			t.Error(err)
		}
	})

	teardown(t, client)
}

func setupAS3(t *testing.T) (client cloud.Client) {
	client = cloud.AS3Client("lockedarchive-test", "us-east-1")
	if err := client.CreateArchive(); err != nil {
		t.Fatal(err)
	}
	t.Log("success: created archive with AS3")
	return
}

func teardown(t *testing.T, client cloud.Client) {
	if err := client.RemoveArchive(); err != nil {
		t.Fatal(err)
	}
	t.Log("success: removed archive with AS3")
}
