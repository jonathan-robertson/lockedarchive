package localstorage

import (
	"os"
	"testing"

	"github.com/puddingfactory/filecabinet/clob"
)

const (
	testCab = "testcabinet"
)

var (
	cache     Cache
	testEntry = clob.Entry{
		Key:       "5678",
		ParentKey: "1234",
		Name:      "test.txt",
		Type:      '-',
	}
)

// TODO: Fix localstorage

func TestNew(t *testing.T) {
	var err error
	if cache, err = Open(testCab); err != nil {
		t.Fatal(err)
	}

	t.Log("cache initialized")
}

func TestRememberEntry(t *testing.T) {
	filename := "localstorage.go"

	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("source file opened")

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("stats received from file")

	entry := testEntry // copy
	entry.Name = filename
	entry.Size = fileInfo.Size()
	entry.LastModified = fileInfo.ModTime()
	entry.Body = file

	if err := cache.RememberEntry(entry); err != nil {
		t.Fatal(err)
	}
	t.Logf("entry with file stats and file data remembered by cache\n%+v\n", entry)
}

func TestRecallEntry(t *testing.T) {
	entry, err := cache.RecallEntry(testEntry.Key)
	if err != nil {
		t.Fatal(err)
	}

	if entry.Type != testEntry.Type {
		t.Fatal("types do not match")
	}

	t.Logf("entry retrieved\n%+v\n", entry)
}

func TestForgetEntry(t *testing.T) {
	if err := cache.ForgetEntry(clob.Entry{Key: testEntry.Key}); err != nil {
		t.Fatal(err)
	}
	t.Log("entry and file data forgotten by cache")
}
