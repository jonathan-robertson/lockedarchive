package s3client

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/puddingfactory/filecabinet/clob"
)

const (
	bucket  = "fcab-test-disk"
	region  = "us-east-1"
	rootKey = "00000000000000000000000000000000"
	testKey = "fe6c582c42c6783085eb68afccf52fa6"
)

var (
	client   Client
	testBody = []byte("This is all I need: a boomfish in my soul.")
)

func init() {
	p := Client{
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
	client, err = New(bucket, region, os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("created bucket", bucket)
}

func TestUpload(t *testing.T) {
	e := clob.Entry{
		Key:          testKey,
		ParentKey:    rootKey,
		Name:         "boom.txt",
		Size:         int64(len(testBody)),
		LastModified: time.Now(),
		Type:         '-',
		Body:         ioutil.NopCloser(bytes.NewReader(testBody)),
	}

	if err := client.Upload(e); err != nil {
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

	if err := client.List("", entries); err != nil {
		t.Fatal(err)
	}

	close(entries)
	<-done

	if entryCount != 1 {
		t.Fatal("expecting 1 entry, got", entryCount)
	}
	t.Log("listed successfully")
}

func TestRename(t *testing.T) {

	// Fetch current metadata of key
	head, err := client.head(testKey)
	if err != nil {
		t.Fatal(err)
	}
	if head.Metadata["Name"] == nil {
		t.Fatal("expected name, but head didn't have one")
	}

	// Build entry based on returned data
	entry, ok := makeEntry(&s3.Object{
		Key:          aws.String(testKey),
		Size:         aws.Int64(0),
		LastModified: aws.Time(time.Now()),
	}, head)
	if !ok {
		t.Fatal("expected entry, but couldn't make one")
	}

	// Update name
	if err := client.Rename(entry, "fish.txt"); err != nil {
		t.Fatal(err)
	}

	// Verify name-change
	if head, err := client.head(testKey); err != nil {
		t.Fatal(err)
	} else if head.Metadata["Name"] == nil {
		t.Fatal("expected name, but head didn't have one")
	} else if *head.Metadata["Name"] != "fish.txt" {
		t.Fatal("expected fish.txt, but got", *head.Metadata["Name"])
	} else {
		t.Log("rename successful")
	}
}

func TestDownload(t *testing.T) {
	var (
		filename = "test.txt"
	)

	// Download to file
	file, err := os.Create(filename)
	if file != nil {
		defer func() {
			if err := os.Remove(filename); err != nil {
				t.Fatal("unable to delete", filename, "once testing concluded")
			}
		}()
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Download(file, clob.Entry{Key: testKey}); err != nil {
		t.Fatal(err)
	}
	file.Close() // done writing

	// Check file's contents
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, testBody) {
		t.Log("download successful")
	} else {
		t.Fatalf("expected \"%s\" but got \"%s\"", testBody, data)
	}
}

func TestDeleteObject(t *testing.T) {
	if err := client.Delete(clob.Entry{Key: testKey}); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted key", testKey)
}

func TestDeleteBucket(t *testing.T) {
	if err := client.DeleteCabinet(); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted bucket", bucket)
}
