package stream_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

// caller responsiblef for closing files
func setup(t *testing.T, srcName, dstName string) (src, dst *os.File) {
	var err error

	// Open file for reading
	if src, err = os.Open(srcName); err != nil {
		t.Fatal(err)
	}

	// Open file for writing
	if dst, err = os.Create(dstName); err != nil {
		t.Fatal(err)
	}

	return
}

func compareAndCleanup(t *testing.T, srcFilename, wrkFilename, dstFilename string) {
	srcData, err := ioutil.ReadFile(srcFilename)
	if err != nil {
		t.Fatal(err)
	}

	dstData, err := ioutil.ReadFile(dstFilename)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(srcData, dstData) {
		t.Fatalf("%s and %s do not match", srcFilename, dstFilename)
	}

	t.Log("source and destination match, as expected")

	if err := os.Remove(wrkFilename); err != nil {
		t.Errorf("trouble removing %s", wrkFilename)
	}

	if err := os.Remove(dstFilename); err != nil {
		t.Errorf("trouble removing %s", dstFilename)
	}
}
