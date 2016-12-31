package localstorage

import (
	"testing"

	"github.com/puddingfactory/filecabinet/clob"
)

const (
	testCab = "testcabinet"
)

var (
	cache = Cache{Cabinet: testCab}
)

func TestNew(t *testing.T) {
	var err error
	if cache, err = New(testCab); err != nil {
		t.Fatal(err)
	}
}

func TestForgetEntry(t *testing.T) {
	if err := cache.ForgetEntry(clob.Entry{Key: "1234"}); err != nil {
		t.Fatal(err)
	}
}
