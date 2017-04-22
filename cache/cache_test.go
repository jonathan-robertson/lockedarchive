package cache

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

type FileContents struct {
	ID       string
	ParentID string
	Name     string
	Body     string
	ModTime  int64 // unix
	IsDir    bool
	Size     int64
}

const (
	testDatabaseName = "test.db"
	testBucketName   = "mybkt"
	testKey          = "testKey"
)

var (
	fcon = FileContents{
		ID:       "testID",
		ParentID: "parentID",
		Name:     "testObject",
		ModTime:  time.Now().Unix(),
		IsDir:    false,
		Size:     29948,
	}
)

func (fc FileContents) MarshalBinary() ([]byte, error) {
	// A simple encoding: plain text.
	var b bytes.Buffer
	fmt.Fprintln(&b, fc.ID, fc.ParentID, fc.Name, fc.ModTime, fc.IsDir, fc.Size)
	return b.Bytes(), nil
}

// UnmarshalBinary modifies the receiver so it must take a pointer receiver.
func (fc *FileContents) UnmarshalBinary(data []byte) error {
	// A simple encoding: plain text.
	b := bytes.NewBuffer(data)
	_, err := fmt.Fscanln(b, &fc.ID, &fc.ParentID, &fc.Name, &fc.ModTime, &fc.IsDir, &fc.Size)
	return err
}

func setup(t *testing.T) {
	if err := Open(testDatabaseName, testBucketName); err != nil {
		t.Fatal(err)
	}
}

func cleanup(t *testing.T) {
	if err := Close(); err != nil {
		t.Fatal(err)
	}
}

func Test(t *testing.T) {
	setup(t)
	defer cleanup(t)

	t.Run("Put", func(t *testing.T) {
		if err := Put(testKey, fcon); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		var fileContents FileContents
		if err := Get(testKey, &fileContents); err != nil {
			t.Fatal(err)
		}
		if fcon != fileContents {
			t.Fatalf("data received from cache doesn't match data put into it\n1. %+v\n2. %+v", fcon, fileContents)
		}
	})
}
