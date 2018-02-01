package cloud_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/cloud"
	"github.com/jonathan-robertson/lockedarchive/secure"
)

const (
	testPassphrase = "passphrase"
	testFilename   = "src.txt"
)

func TestAS3(t *testing.T) {
	secure.Reset()

	client := setupAS3(t)
	entry := cloud.Entry{
		ID:    "1234",
		IsDir: true,
	}
	otherEntry := cloud.Entry{ID: "4567"}

	pc, err := secure.ProtectPassphrase([]byte(testPassphrase))
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Destroy()

	meta, err := entry.Meta(pc)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(testFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := client.Upload(entry.ID, meta, file); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	t.Run("Head", func(t *testing.T) {
		meta, err := client.Head(entry)
		if err != nil {
			t.Error(err)
		}

		sc, err := secure.DecryptWithSaltFromStringToSecret(pc, meta)
		if err != nil {
			t.Error(err)
		}
		defer sc.Destroy()

		if err := json.Unmarshal(sc.Buffer(), &otherEntry); err != nil {
			t.Error(err)
		}
		if otherEntry.IsDir != true {
			t.Error("data unmarshalled to otherEntry doesn't match")
		}
	})
	t.Run("Download", func(t *testing.T) {
		rc, err := client.Download(entry)
		if err != nil {
			t.Error(err)
		}
		if rc != nil {
			if err := rc.Close(); err != nil {
				t.Error(err)
			}
		}
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
