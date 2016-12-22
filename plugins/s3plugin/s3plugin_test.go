package s3plugin

import (
	"log"
	"os"
	"testing"

	"golang.org/x/time/rate"
)

const (
	bucket = "fcab-test-disk"
	region = "us-east-1"
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

func TestDeleteBucket(t *testing.T) {
	if err := plugin.DeleteCabinet(); err != nil {
		t.Fatal(err)
	}
	t.Log("deleted", bucket)
}
