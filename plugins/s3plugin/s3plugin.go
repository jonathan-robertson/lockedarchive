package s3plugin

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

type Plugin struct {
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

// New will return a new s3 plugin after creating the Bucket (if necessary)
func New(cabinetName, region, accessKey, secretKey string) (p Plugin, err error) {
	// REVIEW: Verify validity of cabinetName as a bucket name first
	p = Plugin{
		Bucket:            cabinetName,
		region:            region,
		accessKey:         accessKey,
		secretKey:         secretKey,
		putListDelLimiter: rate.NewLimiter(putListDelLimit, 1),
		getLimiter:        rate.NewLimiter(getLimit, 1),
	}

	if !p.bucketExists() {
		err = p.CreateCabinet()
	}

	return
}

func (p Plugin) bucketExists() bool {
	params := &s3.HeadBucketInput{
		Bucket: aws.String(p.Bucket), // Required
	}

	time.Sleep(p.putListDelLimiter.Reserve().Delay())
	if _, err := p.svc().HeadBucket(params); err != nil {
		return false
	}

	return true
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
		u.PartSize = uploadPartSize
		u.Concurrency = uploadConcurrency
	})
}

func (p Plugin) downloader() *s3manager.Downloader {
	return s3manager.NewDownloaderWithClient(p.svc(), func(d *s3manager.Downloader) {
		d.PartSize = downloadPartSize
		d.Concurrency = downloadConcurrency
	})
}

// CreateCabinet will create a new cabinet in storage
func (p Plugin) CreateCabinet() error {
	params := &s3.CreateBucketInput{
		Bucket: aws.String(p.Bucket),
	}

	if _, err := p.svc().CreateBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return p.confirmBucketCreation()
}

// DeleteCabinet will delete an existing cabinet from storage
func (p Plugin) DeleteCabinet() error {
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(p.Bucket),
	}

	if _, err := p.svc().DeleteBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return p.confirmBucketDeletion()
}

// List fetches keys from storage and translates them to entries, then feeds entries to channel
func (p Plugin) List(prefix string, entries chan clob.Entry) error {
	var (
		wg      sync.WaitGroup
		objects = make(chan *s3.Object, downloadConcurrency)
		params  = &s3.ListObjectsV2Input{
			Bucket: aws.String(p.Bucket), // Required
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
				if header, err := p.head(object); err != nil {
					fmt.Println(err)
				} else {
					if entry, ok := makeEntry(object, header); ok {
						entries <- entry
					}
				}
			}
		}()
	}

	time.Sleep(p.putListDelLimiter.Reserve().Delay())
	err := p.svc().ListObjectsV2Pages(params, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, object := range page.Contents {
			objects <- object
		}

		time.Sleep(p.putListDelLimiter.Reserve().Delay())
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

func (p Plugin) head(object *s3.Object) (*s3.HeadObjectOutput, error) {
	// for object := range objects {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(p.Bucket), // Required
		Key:    object.Key,           // Required
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

	time.Sleep(p.getLimiter.Reserve().Delay())
	resp, err := p.svc().HeadObject(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return nil, err
	}

	return resp, nil
}

// Upload object in s3
func (p Plugin) Upload(e clob.Entry) error {
	upParams := &s3manager.UploadInput{
		Bucket:   aws.String(p.Bucket), // Required
		Key:      aws.String(e.Key),    // Required
		Metadata: composeMetadata(e),
		Body:     e.Body,
	}

	_, err := p.uploader().Upload(upParams)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return p.confirmObjectCreation(e.Key)
}

// Rename updates the Name in metadata
func (p Plugin) Rename(e clob.Entry, newName string) error {
	e.Name = newName // Update Name

	params := &s3.CopyObjectInput{
		Bucket:     aws.String(p.Bucket),               // Required
		CopySource: aws.String(p.Bucket + "/" + e.Key), // Required
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

	if _, err := p.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func (p Plugin) Update(e clob.Entry) error {
	return nil
}

// Delete removes the entry from storage
func (p Plugin) Delete(e clob.Entry) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket), // Required
		Key:    aws.String(e.Key),    // Required
		// MFA:          aws.String("MFA"),
		// RequestPayer: aws.String("RequestPayer"),
		// VersionId:    aws.String("ObjectVersionId"),
	}

	if _, err := p.svc().DeleteObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return p.confirmObjectDeletion(e.Key)
}

// Download writes data from an entry to the provided writer
func (p Plugin) Download(w io.WriterAt, e clob.Entry) error {
	params := &s3.GetObjectInput{
		Bucket: aws.String(p.Bucket), // Required
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

	if _, err := p.downloader().Download(w, params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// Copy duplicates the contents and metadata of one entry to a new key
func (p Plugin) Copy(source clob.Entry, destinationKey string) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(p.Bucket),                    // Required
		CopySource: aws.String(p.Bucket + "/" + source.Key), // Required
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

	if _, err := p.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
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
