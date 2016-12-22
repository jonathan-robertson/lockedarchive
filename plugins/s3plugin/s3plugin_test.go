package s3plugin

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/puddingfactory/filecabinet/clob"
)

const (
	bucket  = "fcab-test-disk"
	region  = "us-east-1"
	rootKey = "00000000000000000000000000000000"
)

var (
	plugin Plugin
)

func init() {
	p := Plugin{
		Bucket:            bucket,
		region:            region,
		accessKey:         os.Getenv("ACCESS_KEY"),
		secretKey:         os.Getenv("SECRET_KEY"),
		putListDelLimiter: rate.NewLimiter(putListDelLimit, 1),
		getLimiter:        rate.NewLimiter(getLimit, 1),
	}

	if p.bucketExists() {
		log.Fatal("Cannot run test since ", bucket, " already exists! Delete this before running again.")
	}
}

// WARNING: The following tests will generate charges to your account and are therefore not recommended for use on every single commit

func TestCreateBucket(t *testing.T) {
	var err error
	plugin, err = New(bucket, region, os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("created", bucket)
}

func TestUpload(t *testing.T) {
	data := []byte("THIS IS ALL I NEED! A FISHY IN MY SOUL")
	e := clob.Entry{
		Key:          generateNewID(),
		ParentKey:    rootKey,
		Name:         "fish.doc",
		Size:         int64(len(data)),
		LastModified: time.Now(),
		Type:         '-',
		Body:         bytes.NewReader(data),
	}

	if err := plugin.Upload(e); err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded", e.Name)
}

func TestDeleteBucket(t *testing.T) {
	if err := plugin.DeleteCabinet(); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted", bucket)
}

func generateNewID() (newID string) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)
}
