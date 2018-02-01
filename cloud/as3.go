package cloud

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"

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

const (
	metadataPrefix = "X-Amz-Meta-La"
)

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
func (client AS3) RemoveArchive() error {
	input := &s3.DeleteBucketInput{
		Bucket: aws.String(client.Bucket),
	}
	_, err := client.svc().DeleteBucket(input)
	// TODO: current design here will error if there are any objcts within bucket
	// Perhaps we want this behavior for now, exposing another func for delete all obj
	// or maybe not. bool could be passed into RemoveArchive to confirm if we want to
	// delete containing objects
	return evalErr(err)
}

// List collects all list data for the given bucket; closes Entry chan when done
func (client AS3) List(entries chan Entry) error {
	defer close(entries)
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(client.Bucket),
	}
	return evalErr(client.svc().ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			entries <- Entry{
				Key:          aws.StringValue(obj.Key),
				Size:         aws.Int64Value(obj.Size),
				LastModified: aws.TimeValue(obj.LastModified),
			}
		}
		return aws.BoolValue(page.IsTruncated)
	}))
}

// Upload sends an Entry to S3, along with its body and properties
func (client AS3) Upload(id, metadata string, file *os.File) error {
	hash, err := makeMD5(file)
	if err != nil {
		return err
	}

	metaMap := make(map[string]string)
	metaMap[metadataPrefix] = metadata

	input := &s3.PutObjectInput{
		Bucket:     aws.String(client.Bucket),
		Key:        aws.String(id),
		Metadata:   aws.StringMap(metaMap),
		ContentMD5: aws.String(hash),
		// Tagging: aws.String("key1=value1&key2=value2"), // TODO: add this in later?
	}

	if file != nil {
		input.Body = aws.ReadSeekCloser(file)
	}

	_, err = client.svc().PutObject(input)
	return evalErr(err)
}

func makeMD5(file *os.File) (hash64 string, err error) {
	if file == nil {
		return "", nil
	}

	// Hash contents of file
	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return
	}
	hashSum := hash.Sum(nil)

	// Encode to base64
	hash64 = base64.StdEncoding.EncodeToString(hashSum)
	return
}

// Download fetches entry's data from S3 and Puts it in cache
func (client AS3) Download(entry Entry) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(client.Bucket),
		Key:    aws.String(entry.ID),
	}

	result, err := client.svc().GetObject(input)
	if err != nil {
		return nil, evalErr(err)
	}

	if aws.Int64Value(result.ContentLength) == 0 {
		// TODO: return error? does this even matter? Dirs will be size 0...
	}

	// TODO: Confirm that local checksum matches ETAG
	// NOTE: md5 checksums are sent to AS3 in bytes -> base64 format,
	// while ETags are returned as bytes -> hex format, wrapped in quotes.
	// Storing the bytes form of checksum locally and comparing it is probably a good idea.
	// Maybe a check like this is better suited to Head method - to invalidate cached body
	// for the file in the Head request.

	return result.Body, nil
}

// Head fetches metadata information for an entry
func (client AS3) Head(entry Entry) (string, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(client.Bucket),
		Key:    aws.String(entry.ID),
	}

	result, err := client.svc().HeadObject(input)
	if err != nil {
		return "", evalErr(err)
	}

	// entry.LastModified = aws.TimeValue(result.LastModified)
	// TODO: if result.ETag differs from local checksum, remove cached version
	// TODO: receive tags

	// fmt.Printf("head: %+v\n", result)
	// Metadata map[string]*string `location:"headers" locationName:"x-amz-meta-" type:"map"`

	metadata := aws.StringValue(result.Metadata[metadataPrefix])
	return metadata, nil
}

// Update sends
func (client AS3) Update(entry Entry) error {

	return nil
}

// Delete removes an Entry from S3
func (client AS3) Delete(entry Entry) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(client.Bucket),
		Key:    aws.String(entry.ID),
	}

	_, err := client.svc().DeleteObject(input)
	err = evalErr(err)
	return err
}

func (client AS3) svc() *s3.S3 {
	return s3.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
}

// evalErr surfaces more info from an error and returns it
func evalErr(err error) error {
	if err == nil {
		return nil
	}

	// TODO: type switch instead of this??

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeBucketAlreadyExists:
			return fmt.Errorf("%s: %s", s3.ErrCodeBucketAlreadyExists, aerr.Error())
		case s3.ErrCodeBucketAlreadyOwnedByYou:
			return fmt.Errorf("%s: %s", s3.ErrCodeBucketAlreadyOwnedByYou, aerr.Error())
		case s3.ErrCodeNoSuchKey:
			return fmt.Errorf("%s: %s", s3.ErrCodeNoSuchKey, aerr.Error())
		default:
			return aerr
		}
	}

	return err
}
