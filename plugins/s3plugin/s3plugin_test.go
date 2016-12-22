package s3plugin

import (
	"bytes"
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
	testKey = "fe6c582c42c6783085eb68afccf52fa6"
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
	t.Log("created bucket", bucket)
}

func TestUpload(t *testing.T) {
	data := []byte("THIS IS ALL I NEED! A FISHY IN MY SOUL")
	e := clob.Entry{
		Key:          testKey,
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
	t.Log("uploaded key", testKey)
}

func TestList(t *testing.T) {
	var (
		entries    = make(chan clob.Entry)
		done       = make(chan bool)
		entryCount int
	)

	go func() {
		defer close(done)
		for entry := range entries {
			entryCount++
			t.Logf("%+v\n", entry)
		}
	}()

	if err := plugin.List("", entries); err != nil {
		t.Fatal(err)
	}

	close(entries)
	<-done

	if entryCount != 1 {
		t.Fatal("expecting 1 entry, got", entryCount)
	}
	t.Log("listed successfully")
}

func TestDeleteObject(t *testing.T) {
	if err := plugin.Delete(clob.Entry{Key: testKey}); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted key", testKey)
}

func TestDeleteBucket(t *testing.T) {
	if err := plugin.DeleteCabinet(); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted bucket", bucket)
}
