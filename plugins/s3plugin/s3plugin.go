package s3plugin

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/puddingfactory/filecabinet/clob"
)

type Plugin struct {
	Bucket, region, accessKey, secretKey string
}

var (
	UploadPartSize      int64 = 64 * 1024 * 1024 // 64MB
	DownloadPartSize    int64 = 64 * 1024 * 1024 // 64MB
	UploadConcurrency         = 10
	DownloadConcurrency       = 10
)

// New will return a new s3 plugin after creating the Bucket (if necessary)
func New(cabinetName, region, accessKey, secretKey string) (p Plugin, err error) {
	// REVIEW: Verify validity of cabinetName as a bucket name first
	p = Plugin{
		Bucket:    cabinetName,
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
	}

	// TODO: does bucket exist?

	err = p.CreateCabinet()
	return
}

func (p Plugin) svc() *s3.S3 {
	token := "" // unsure of purpose

	config := &aws.Config{
		Region: aws.String(p.region),
		Credentials: credentials.NewStaticCredentials(
			p.accessKey,
			p.secretKey,
			token),
	}

	return s3.New(session.New(config))
}

func (p Plugin) uploader() *s3manager.Uploader {
	return s3manager.NewUploaderWithClient(p.svc(), func(u *s3manager.Uploader) {
		u.PartSize = UploadPartSize
		u.Concurrency = UploadConcurrency
	})
}

func (p Plugin) downloader() *s3manager.Downloader {
	return s3manager.NewDownloaderWithClient(p.svc(), func(d *s3manager.Downloader) {
		d.PartSize = DownloadPartSize
		d.Concurrency = DownloadConcurrency
	})
}

func (p Plugin) CreateCabinet() error {
	params := &s3.CreateBucketInput{
		Bucket: aws.String(p.Bucket), // Required
		// ACL:    aws.String("BucketCannedACL"),
		// CreateBucketConfiguration: &s3.CreateBucketConfiguration{
		// 	LocationConstraint: aws.String("BucketLocationConstraint"),
		// },
		// GrantFullControl: aws.String("GrantFullControl"),
		// GrantRead:        aws.String("GrantRead"),
		// GrantReadACP:     aws.String("GrantReadACP"),
		// GrantWrite:       aws.String("GrantWrite"),
		// GrantWriteACP:    aws.String("GrantWriteACP"),
	}
	resp, err := p.svc().CreateBucket(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(resp)

	return p.confirmBucketCreation()
}

func (p Plugin) DeleteCabinet() error {
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(p.Bucket), // Required
	}
	resp, err := p.svc().DeleteBucket(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(resp)

	return p.confirmBucketDeletion()
}

func (p Plugin) List(prefix string, entries chan clob.Entry) error {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(p.Bucket), // Required
		// ContinuationToken: aws.String("Token"),
		// Delimiter:         aws.String("Delimiter"),
		// EncodingType:      aws.String("EncodingType"),
		// FetchOwner:        aws.Bool(true),
		// MaxKeys:           aws.Int64(1),
		Prefix: aws.String(prefix),
		// RequestPayer:      aws.String("RequestPayer"),
		// StartAfter:        aws.String("StartAfter"),
	}

	err := p.svc().ListObjectsV2Pages(params, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, o := range page.Contents {
			fmt.Printf("%+v\n", o) // TODO: Test and update
			// key := strings.Split(*o.Key, "/")
			// entries <- clob.Entry{
			// 	ParentID: key[0],
			// 	ID:       key[1],
			// 	// Metadata: , // TODO
			// }
		}

		return *page.IsTruncated // REVIEW: This may not be totally accurate at all times
	})

	return err
}

// Create new object in s3
func (p Plugin) Create(e clob.Entry) error {
	upParams := &s3manager.UploadInput{
		Bucket:   aws.String(p.Bucket), // Required
		Key:      aws.String(e.Key()),  // Required
		Metadata: convMetaToAWS(e.Metadata),
		Body:     e.Body,
	}

	result, err := p.uploader().Upload(upParams)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(result)
	return p.confirmObjectCreation(e.Key())
}

// Rename updates the Name in metadata
func (p Plugin) Rename(e clob.Entry, newName string) error {
	e.Metadata["name"] = newName // Update Name

	params := &s3.CopyObjectInput{
		Bucket:     aws.String(p.Bucket), // Required
		CopySource: aws.String(e.Key()),  // Required
		Key:        aws.String(e.Key()),  // Required
		// ACL:                            aws.String("ObjectCannedACL"),
		// CacheControl:                   aws.String("CacheControl"),
		// ContentDisposition:             aws.String("ContentDisposition"),
		// ContentEncoding:                aws.String("ContentEncoding"),
		// ContentLanguage:                aws.String("ContentLanguage"),
		// ContentType:                    aws.String("ContentType"),
		// CopySourceIfMatch:              aws.String("CopySourceIfMatch"),
		// CopySourceIfModifiedSince:      aws.Time(time.Now()),
		// CopySourceIfNoneMatch:          aws.String("CopySourceIfNoneMatch"),
		// CopySourceIfUnmodifiedSince:    aws.Time(time.Now()),
		// CopySourceSSECustomerAlgorithm: aws.String("CopySourceSSECustomerAlgorithm"),
		// CopySourceSSECustomerKey:       aws.String("CopySourceSSECustomerKey"),
		// CopySourceSSECustomerKeyMD5:    aws.String("CopySourceSSECustomerKeyMD5"),
		// Expires:                        aws.Time(time.Now()),
		// GrantFullControl:               aws.String("GrantFullControl"),
		// GrantRead:                      aws.String("GrantRead"),
		// GrantReadACP:                   aws.String("GrantReadACP"),
		// GrantWriteACP:                  aws.String("GrantWriteACP"),
		Metadata:          convMetaToAWS(e.Metadata),
		MetadataDirective: aws.String(s3.MetadataDirectiveReplace),
		// RequestPayer:            aws.String("RequestPayer"),
		// SSECustomerAlgorithm:    aws.String("SSECustomerAlgorithm"),
		// SSECustomerKey:          aws.String("SSECustomerKey"),
		// SSECustomerKeyMD5:       aws.String("SSECustomerKeyMD5"),
		// SSEKMSKeyId:             aws.String("SSEKMSKeyId"),
		// ServerSideEncryption:    aws.String("ServerSideEncryption"),
		// StorageClass:            aws.String("StorageClass"),
		// WebsiteRedirectLocation: aws.String("WebsiteRedirectLocation"),
	}
	resp, err := p.svc().CopyObject(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(resp)
	return nil
}

func (p Plugin) Update(e clob.Entry) error {
	return nil
}

// Delete removes the entry from storage
func (p Plugin) Delete(e clob.Entry) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket), // Required
		Key:    aws.String(e.Key()),  // Required
		// MFA:          aws.String("MFA"),
		// RequestPayer: aws.String("RequestPayer"),
		// VersionId:    aws.String("ObjectVersionId"),
	}
	resp, err := p.svc().DeleteObject(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(resp)
	return p.confirmObjectDeletion(e.Key())
}

// Download writes data from an entry to the provided writer
func (p Plugin) Download(w io.WriterAt, e clob.Entry) error {
	params := &s3.GetObjectInput{
		Bucket: aws.String(p.Bucket), // Required
		Key:    aws.String(e.Key()),  // Required
		// IfMatch:                    aws.String("IfMatch"),
		// IfModifiedSince:            aws.Time(time.Now()),
		// IfNoneMatch:                aws.String("IfNoneMatch"),
		// IfUnmodifiedSince:          aws.Time(time.Now()),
		// PartNumber:                 aws.Int64(1),
		// Range:                      aws.String("Range"),
		// RequestPayer:               aws.String("RequestPayer"),
		// ResponseCacheControl:       aws.String("ResponseCacheControl"),
		// ResponseContentDisposition: aws.String("ResponseContentDisposition"),
		// ResponseContentEncoding:    aws.String("ResponseContentEncoding"),
		// ResponseContentLanguage:    aws.String("ResponseContentLanguage"),
		// ResponseContentType:        aws.String("ResponseContentType"),
		// ResponseExpires:            aws.Time(time.Now()),
		// SSECustomerAlgorithm:       aws.String("SSECustomerAlgorithm"),
		// SSECustomerKey:             aws.String("SSECustomerKey"),
		// SSECustomerKeyMD5:          aws.String("SSECustomerKeyMD5"),
		// VersionId:                  aws.String("ObjectVersionId"),
	}
	n, err := p.downloader().Download(w, params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(n, "bytes written")
	return nil
}

// Copy duplicates the contents and metadata of one entry to a new key
func (p Plugin) Copy(source clob.Entry, destinationKey string) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(p.Bucket),       // Required
		CopySource: aws.String(source.Key()),   // Required
		Key:        aws.String(destinationKey), // Required
		// ACL:                            aws.String("ObjectCannedACL"),
		// CacheControl:                   aws.String("CacheControl"),
		// ContentDisposition:             aws.String("ContentDisposition"),
		// ContentEncoding:                aws.String("ContentEncoding"),
		// ContentLanguage:                aws.String("ContentLanguage"),
		// ContentType:                    aws.String("ContentType"),
		// CopySourceIfMatch:              aws.String("CopySourceIfMatch"),
		// CopySourceIfModifiedSince:      aws.Time(time.Now()),
		// CopySourceIfNoneMatch:          aws.String("CopySourceIfNoneMatch"),
		// CopySourceIfUnmodifiedSince:    aws.Time(time.Now()),
		// CopySourceSSECustomerAlgorithm: aws.String("CopySourceSSECustomerAlgorithm"),
		// CopySourceSSECustomerKey:       aws.String("CopySourceSSECustomerKey"),
		// CopySourceSSECustomerKeyMD5:    aws.String("CopySourceSSECustomerKeyMD5"),
		// Expires:                        aws.Time(time.Now()),
		// GrantFullControl:               aws.String("GrantFullControl"),
		// GrantRead:                      aws.String("GrantRead"),
		// GrantReadACP:                   aws.String("GrantReadACP"),
		// GrantWriteACP:                  aws.String("GrantWriteACP"),
		// Metadata:          convMetaToAWS(source.Metadata),
		MetadataDirective: aws.String(s3.MetadataDirectiveCopy),
		// RequestPayer:            aws.String("RequestPayer"),
		// SSECustomerAlgorithm:    aws.String("SSECustomerAlgorithm"),
		// SSECustomerKey:          aws.String("SSECustomerKey"),
		// SSECustomerKeyMD5:       aws.String("SSECustomerKeyMD5"),
		// SSEKMSKeyId:             aws.String("SSEKMSKeyId"),
		// ServerSideEncryption:    aws.String("ServerSideEncryption"),
		// StorageClass:            aws.String("StorageClass"),
		// WebsiteRedirectLocation: aws.String("WebsiteRedirectLocation"),
	}
	resp, err := p.svc().CopyObject(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	// Pretty-print the response data.
	fmt.Println(resp)
	return p.confirmObjectCreation(destinationKey)
}

func (p Plugin) confirmBucketCreation() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(p.Bucket),
	}
	return p.svc().WaitUntilBucketExists(headBucketInput)
}

func (p Plugin) confirmBucketDeletion() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(p.Bucket),
	}
	return p.svc().WaitUntilBucketNotExists(headBucketInput)
}

func (p Plugin) confirmObjectCreation(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	}
	return p.svc().WaitUntilObjectExists(headObjectInput)
}

func (p Plugin) confirmObjectDeletion(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	}
	return p.svc().WaitUntilObjectNotExists(headObjectInput)
}

// convMetaToAWS converts entry metadata to aws metadata
func convMetaToAWS(entryMeta map[string]string) map[string]*string {
	awsMeta := make(map[string]*string)
	for k, v := range entryMeta {
		awsMeta[k] = aws.String(v)
	}
	return awsMeta
}

// convMetaToEntry converts aws metadata to entry metadata
func convMetaToEntry(awsMeta map[string]*string) map[string]string {
	entryMeta := make(map[string]string)
	for k, v := range awsMeta {
		entryMeta[k] = *v
	}
	return entryMeta
}
