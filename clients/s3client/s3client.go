package s3client

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/puddingfactory/filecabinet/clob"
)

// Client represents the structure for s3 client
type Client struct {
	Bucket            string
	region            string
	accessKey         string
	secretKey         string
	putListDelLimiter *rate.Limiter
	getLimiter        *rate.Limiter
}

var (
	uploadPartSize      int64 = 64 * 1024 * 1024 // 64MB
	downloadPartSize    int64 = 64 * 1024 * 1024 // 64MB
	uploadConcurrency         = 10
	downloadConcurrency       = 10

	putListDelLimit rate.Limit = 100
	getLimit        rate.Limit = 300
)

// New will return a new s3 client after creating the Bucket (if necessary)
func New(cabinetName, region, accessKey, secretKey string) (c Client, err error) {
	// REVIEW: Verify validity of cabinetName as a bucket name first
	c = Client{
		Bucket:            cabinetName,
		region:            region,
		accessKey:         accessKey,
		secretKey:         secretKey,
		putListDelLimiter: rate.NewLimiter(putListDelLimit, 1),
		getLimiter:        rate.NewLimiter(getLimit, 1),
	}

	if !c.bucketExists() {
		err = c.CreateCabinet()
	}

	return
}

func (c Client) bucketExists() bool {
	params := &s3.HeadBucketInput{
		Bucket: aws.String(c.Bucket), // Required
	}

	time.Sleep(c.putListDelLimiter.Reserve().Delay())
	if _, err := c.svc().HeadBucket(params); err != nil {
		return false
	}

	return true
}

func (c Client) svc() *s3.S3 {
	token := "" // unsure of purpose

	config := &aws.Config{
		Region: aws.String(c.region),
		Credentials: credentials.NewStaticCredentials(
			c.accessKey,
			c.secretKey,
			token),
	}

	return s3.New(session.New(config))
}

func (c Client) uploader() *s3manager.Uploader {
	return s3manager.NewUploaderWithClient(c.svc(), func(u *s3manager.Uploader) {
		u.PartSize = uploadPartSize
		u.Concurrency = uploadConcurrency
	})
}

func (c Client) downloader() *s3manager.Downloader {
	return s3manager.NewDownloaderWithClient(c.svc(), func(d *s3manager.Downloader) {
		d.PartSize = downloadPartSize
		d.Concurrency = downloadConcurrency
	})
}

// CreateCabinet will create a new cabinet in storage
func (c Client) CreateCabinet() error {
	params := &s3.CreateBucketInput{
		Bucket: aws.String(c.Bucket),
	}

	if _, err := c.svc().CreateBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return c.confirmBucketCreation()
}

// DeleteCabinet will delete an existing cabinet from storage
func (c Client) DeleteCabinet() error {
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(c.Bucket),
	}

	if _, err := c.svc().DeleteBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return c.confirmBucketDeletion()
}

// List fetches keys from storage and translates them to entries, then feeds entries to channel
func (c Client) List(prefix string, entries chan clob.Entry) error {
	var (
		wg      sync.WaitGroup
		objects = make(chan *s3.Object, downloadConcurrency)
		params  = &s3.ListObjectsV2Input{
			Bucket: aws.String(c.Bucket), // Required
			// ContinuationToken: aws.String("Token"),
			// Delimiter:         aws.String("Delimiter"),
			EncodingType: aws.String("url"),
			// FetchOwner:        aws.Bool(true),
			// MaxKeys:           aws.Int64(1),
			Prefix: aws.String(prefix),
			// RequestPayer:      aws.String("RequestPayer"),
			// StartAfter:        aws.String("StartAfter"),
		}
	)

	for i := 0; i < downloadConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for object := range objects {
				if header, err := c.head(*object.Key); err != nil {
					fmt.Println(err)
				} else {
					if entry, ok := makeEntry(object, header); ok {
						entries <- entry
					}
				}
			}
		}()
	}

	time.Sleep(c.putListDelLimiter.Reserve().Delay())
	err := c.svc().ListObjectsV2Pages(params, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, object := range page.Contents {
			objects <- object
		}

		time.Sleep(c.putListDelLimiter.Reserve().Delay())
		return *page.IsTruncated // REVIEW: This may not be totally accurate at all times
	})

	close(objects)
	wg.Wait()

	return err
}

func extractRuneHelper(str string) (r rune) {
	for _, r = range str { // extracts first character to rune and returns
		break
	}
	return
}

func makeEntry(object *s3.Object, head *s3.HeadObjectOutput) (entry clob.Entry, success bool) {
	if head.Metadata["Type"] == nil || head.Metadata["Parent-Key"] == nil || head.Metadata["Name"] == nil {
		fmt.Printf("ERROR: Could not process as entry: object: %+v, header: %+v\n", object, head)
		return
	}

	return clob.Entry{
		Key:          *object.Key,
		ParentKey:    *head.Metadata["Parent-Key"],
		Name:         *head.Metadata["Name"],
		Size:         *object.Size,
		LastModified: *object.LastModified,
		Type:         extractRuneHelper(*head.Metadata["Type"]),
	}, true
}

func composeMetadata(e clob.Entry) (metadata map[string]*string) {
	return map[string]*string{
		"parent-key": aws.String(e.ParentKey),
		"name":       aws.String(e.Name),
		"type":       aws.String(fmt.Sprintf("%c", e.Type)),
	}
}

func (c Client) head(key string) (*s3.HeadObjectOutput, error) {
	// for object := range objects {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(c.Bucket), // Required
		Key:    aws.String(key),      // Required
		// IfMatch:              aws.String("IfMatch"),
		// IfModifiedSince:      aws.Time(time.Now()),
		// IfNoneMatch:          aws.String("IfNoneMatch"),
		// IfUnmodifiedSince:    aws.Time(time.Now()),
		// PartNumber:           aws.Int64(1),
		// Range:                aws.String("Range"),
		// RequestPayer:         aws.String("RequestPayer"),
		// SSECustomerAlgorithm: aws.String("SSECustomerAlgorithm"),
		// SSECustomerKey:       aws.String("SSECustomerKey"),
		// SSECustomerKeyMD5:    aws.String("SSECustomerKeyMD5"),
		// VersionId:            aws.String("ObjectVersionId"),
	}

	time.Sleep(c.getLimiter.Reserve().Delay())
	resp, err := c.svc().HeadObject(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return nil, err
	}

	return resp, nil
}

// Upload object in s3
func (c Client) Upload(e clob.Entry) error {
	upParams := &s3manager.UploadInput{
		Bucket:   aws.String(c.Bucket), // Required
		Key:      aws.String(e.Key),    // Required
		Metadata: composeMetadata(e),
		Body:     e.Body,
	}

	if _, err := c.uploader().Upload(upParams); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return c.confirmObjectCreation(e.Key)
}

// Move updates the ParentKey in metadata
func (c Client) Move(e clob.Entry, newParent string) error {
	e.ParentKey = newParent
	return c.Update(e)
}

// Rename updates the Name in metadata
func (c Client) Rename(e clob.Entry, newName string) error {
	e.Name = newName // Update Name
	return c.Update(e)
}

// Update pushes changes in metadata to the online object
func (c Client) Update(e clob.Entry) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(c.Bucket),               // Required
		CopySource: aws.String(c.Bucket + "/" + e.Key), // Required
		Key:        aws.String(e.Key),                  // Required
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
		Metadata:          composeMetadata(e),
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

	if _, err := c.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// Delete removes the entry from storage
func (c Client) Delete(e clob.Entry) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(c.Bucket), // Required
		Key:    aws.String(e.Key),    // Required
		// MFA:          aws.String("MFA"),
		// RequestPayer: aws.String("RequestPayer"),
		// VersionId:    aws.String("ObjectVersionId"),
	}

	if _, err := c.svc().DeleteObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return c.confirmObjectDeletion(e.Key)
}

// Download writes data from an entry to the provided writer
func (c Client) Download(w io.WriterAt, e clob.Entry) error {
	params := &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket), // Required
		Key:    aws.String(e.Key),    // Required
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

	if _, err := c.downloader().Download(w, params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// OpenDownstream links the object data with entry.Body
func (c Client) OpenDownstream(e *clob.Entry) (err error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket), // Required
		Key:    aws.String(e.Key),    // Required
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
	if resp, err := c.svc().GetObject(params); err == nil {
		e.Body = resp.Body
	}
	return
}

// Copy duplicates the contents and metadata of one entry to a new key
func (c Client) Copy(source clob.Entry, destinationKey string) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(c.Bucket),                    // Required
		CopySource: aws.String(c.Bucket + "/" + source.Key), // Required
		Key:        aws.String(destinationKey),              // Required
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

	if _, err := c.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return c.confirmObjectCreation(destinationKey)
}

func (c Client) confirmBucketCreation() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(c.Bucket),
	}
	return c.svc().WaitUntilBucketExists(headBucketInput)
}

func (c Client) confirmBucketDeletion() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(c.Bucket),
	}
	return c.svc().WaitUntilBucketNotExists(headBucketInput)
}

func (c Client) confirmObjectCreation(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}
	return c.svc().WaitUntilObjectExists(headObjectInput)
}

func (c Client) confirmObjectDeletion(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}
	return c.svc().WaitUntilObjectNotExists(headObjectInput)
}
