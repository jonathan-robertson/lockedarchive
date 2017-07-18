package cloud

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// AS3 is used to access AWS S3 services in a way that satisfies the Client interface
type AS3 struct {
	Bucket string
	Region string
}

// AS3Client returns a new Client
func AS3Client(bucketName, region string) (client Client) {
	return &AS3{
		Bucket: bucketName,
		Region: region,
	}
}

// CreateArchive creates a new Bucket responsible for storing data
func (client AS3) CreateArchive() error {
	_, err := client.svc().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(client.Bucket),
	})
	return evalErr(err)
}

// RemoveArchive removes the LockedArchive Bucket
// TODO: How should this work? Seems pretty unsafe
func (client *AS3) RemoveArchive() error {
	return nil
}

// List collects all list data for the given bucket
func (client *AS3) List() ([]Entry, error) {
	return nil, nil
}

//
func (client *AS3) Upload(Entry) error {
	return nil
}

//
func (client *AS3) Download(Entry) error {
	return nil
}

//
func (client *AS3) Update(Entry) error {
	return nil
}

//
func (client *AS3) Delete(Entry) error {
	return nil
}

func (client AS3) svc() *s3.S3 {
	return s3.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
}

// evalErr surfaces more info from an error and returns it
func evalErr(err error) error {

	// TODO: type switch instead of this??

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeBucketAlreadyExists:
			return fmt.Errorf("%s: %s", s3.ErrCodeBucketAlreadyExists, aerr.Error())
		case s3.ErrCodeBucketAlreadyOwnedByYou:
			return fmt.Errorf("%s: %s", s3.ErrCodeBucketAlreadyOwnedByYou, aerr.Error())
		default:
			return aerr
		}
	}

	return err
}
