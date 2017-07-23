package cloud

import "testing"

func TestAS3(t *testing.T) {
	var (
		client = setupAS3(t)
		entry  = Entry{
			Key:   "123",
			IsDir: true,
		}
	)

	t.Run("Upload", func(t *testing.T) {
		if err := client.Upload(entry); err != nil {
			t.Error(err)
		}
	})
	t.Run("Head", func(t *testing.T) {
		if err := client.Head(entry); err != nil {
			t.Error(err)
		}
	})
	t.Run("Download", func(t *testing.T) {
		if err := client.Download(entry); err != nil {
			t.Error(err)
		}
	})
	t.Run("List", func(t *testing.T) {
		entries := make(chan Entry)
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

func setupAS3(t *testing.T) (client Client) {
	client = AS3Client("lockedarchive-test", "us-east-1")
	if err := client.CreateArchive(); err != nil {
		t.Fatal(err)
	}
	t.Log("success: created archive with AS3")
	return
}

func teardown(t *testing.T, client Client) {
	if err := client.RemoveArchive(); err != nil {
		t.Fatal(err)
	}
	t.Log("success: removed archive with AS3")
}
