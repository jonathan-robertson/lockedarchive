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
	cache     = Cache{Cabinet: testCab}
	testEntry = clob.Entry{
		Key:       "5678",
		ParentKey: "1234",
		Name:      "test.txt",
		Type:      '-',
	}
)

func TestNew(t *testing.T) {
	var err error
	if cache, err = New(testCab); err != nil {
		t.Fatal(err)
	}
}

func TestRememberEntry(t *testing.T) {
	filename := "localstorage.go"

	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	e := testEntry // copy
	e.Name = filename
	e.Size = fileInfo.Size()
	e.LastModified = fileInfo.ModTime()
	e.Body = file

	if err := cache.RememberEntry(e); err != nil {
		t.Fatal(err)
	}
}

func TestForgetEntry(t *testing.T) {
	if err := cache.ForgetEntry(clob.Entry{Key: testEntry.Key}); err != nil {
		t.Fatal(err)
	}
}
