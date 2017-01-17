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
func New(cabinetName, region, accessKey, secretKey string) (client Client, err error) {
	// REVIEW: Verify validity of cabinetName as a bucket name first
	client = Client{
		Bucket:            cabinetName,
		region:            region,
		accessKey:         accessKey,
		secretKey:         secretKey,
		putListDelLimiter: rate.NewLimiter(putListDelLimit, 1),
		getLimiter:        rate.NewLimiter(getLimit, 1),
	}

	if !client.bucketExists() {
		err = client.CreateCabinet()
	}

	return
}

func (client Client) bucketExists() bool {
	params := &s3.HeadBucketInput{
		Bucket: aws.String(client.Bucket), // Required
	}

	time.Sleep(client.putListDelLimiter.Reserve().Delay())
	if _, err := client.svc().HeadBucket(params); err != nil {
		return false
	}

	return true
}

func (client Client) svc() *s3.S3 {
	token := "" // unsure of purpose

	config := &aws.Config{
		Region: aws.String(client.region),
		Credentials: credentials.NewStaticCredentials(
			client.accessKey,
			client.secretKey,
			token),
	}

	return s3.New(session.New(config))
}

func (client Client) uploader() *s3manager.Uploader {
	return s3manager.NewUploaderWithClient(client.svc(), func(u *s3manager.Uploader) {
		u.PartSize = uploadPartSize
		u.Concurrency = uploadConcurrency
	})
}

func (client Client) downloader() *s3manager.Downloader {
	return s3manager.NewDownloaderWithClient(client.svc(), func(d *s3manager.Downloader) {
		d.PartSize = downloadPartSize
		d.Concurrency = downloadConcurrency
	})
}

// CreateCabinet will create a new cabinet in storage
func (client Client) CreateCabinet() error {
	params := &s3.CreateBucketInput{
		Bucket: aws.String(client.Bucket),
	}

	if _, err := client.svc().CreateBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return client.confirmBucketCreation()
}

// DeleteCabinet will delete an existing cabinet from storage
func (client Client) DeleteCabinet() error {
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(client.Bucket),
	}

	if _, err := client.svc().DeleteBucket(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return client.confirmBucketDeletion()
}

// List fetches keys from storage and translates them to entries, then feeds entries to channel
func (client Client) List(prefix string, entries chan clob.Entry) error {
	var (
		wg      sync.WaitGroup
		objects = make(chan *s3.Object, downloadConcurrency)
		params  = &s3.ListObjectsV2Input{
			Bucket: aws.String(client.Bucket), // Required
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
				if header, err := client.head(*object.Key); err != nil {
					fmt.Println(err)
				} else {
					if entry, ok := makeEntry(object, header); ok {
						entries <- entry
					}
				}
			}
		}()
	}

	time.Sleep(client.putListDelLimiter.Reserve().Delay())
	err := client.svc().ListObjectsV2Pages(params, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, object := range page.Contents {
			objects <- object
		}

		time.Sleep(client.putListDelLimiter.Reserve().Delay())
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

func composeMetadata(entry clob.Entry) (metadata map[string]*string) {
	return map[string]*string{
		"parent-key": aws.String(entry.ParentKey),
		"name":       aws.String(entry.Name),
		"type":       aws.String(fmt.Sprintf("%c", entry.Type)),
	}
}

func (client Client) head(key string) (*s3.HeadObjectOutput, error) {
	// for object := range objects {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(client.Bucket), // Required
		Key:    aws.String(key),           // Required
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

	time.Sleep(client.getLimiter.Reserve().Delay())
	resp, err := client.svc().HeadObject(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return nil, err
	}

	return resp, nil
}

// Upload object in s3
func (client Client) Upload(entry clob.Entry) error {
	upParams := &s3manager.UploadInput{
		Bucket:   aws.String(client.Bucket), // Required
		Key:      aws.String(entry.Key),     // Required
		Metadata: composeMetadata(entry),
		Body:     entry.Body,
	}

	if _, err := client.uploader().Upload(upParams); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return client.confirmObjectCreation(entry.Key)
}

// Move updates the ParentKey in metadata
func (client Client) Move(entry clob.Entry, newParent string) error {
	entry.ParentKey = newParent
	return client.Update(entry)
}

// Rename updates the Name in metadata
func (client Client) Rename(entry clob.Entry, newName string) error {
	entry.Name = newName // Update Name
	return client.Update(entry)
}

// Update pushes changes in metadata to the online object
func (client Client) Update(entry clob.Entry) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(client.Bucket),                   // Required
		CopySource: aws.String(client.Bucket + "/" + entry.Key), // Required
		Key:        aws.String(entry.Key),                       // Required
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
		Metadata:          composeMetadata(entry),
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

	if _, err := client.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// Delete removes the entry from storage
func (client Client) Delete(entry clob.Entry) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(client.Bucket), // Required
		Key:    aws.String(entry.Key),     // Required
		// MFA:          aws.String("MFA"),
		// RequestPayer: aws.String("RequestPayer"),
		// VersionId:    aws.String("ObjectVersionId"),
	}

	if _, err := client.svc().DeleteObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return client.confirmObjectDeletion(entry.Key)
}

// Download writes data from an entry to the provided writer
func (client Client) Download(w io.WriterAt, entry clob.Entry) error {
	params := &s3.GetObjectInput{
		Bucket: aws.String(client.Bucket), // Required
		Key:    aws.String(entry.Key),     // Required
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

	if _, err := client.downloader().Download(w, params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// OpenDownstream links the object data with entry.Body
func (client Client) OpenDownstream(entry *clob.Entry) (err error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(client.Bucket), // Required
		Key:    aws.String(entry.Key),     // Required
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
	if resp, err := client.svc().GetObject(params); err == nil {
		entry.Body = resp.Body
	}
	return
}

// Copy duplicates the contents and metadata of one entry to a new key
func (client Client) Copy(source clob.Entry, destinationKey string) error {
	params := &s3.CopyObjectInput{
		Bucket:     aws.String(client.Bucket),                    // Required
		CopySource: aws.String(client.Bucket + "/" + source.Key), // Required
		Key:        aws.String(destinationKey),                   // Required
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

	if _, err := client.svc().CopyObject(params); err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}
	return client.confirmObjectCreation(destinationKey)
}

func (client Client) confirmBucketCreation() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(client.Bucket),
	}
	return client.svc().WaitUntilBucketExists(headBucketInput)
}

func (client Client) confirmBucketDeletion() error {
	headBucketInput := &s3.HeadBucketInput{
		Bucket: aws.String(client.Bucket),
	}
	return client.svc().WaitUntilBucketNotExists(headBucketInput)
}

func (client Client) confirmObjectCreation(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(client.Bucket),
		Key:    aws.String(key),
	}
	return client.svc().WaitUntilObjectExists(headObjectInput)
}

func (client Client) confirmObjectDeletion(key string) error {
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(client.Bucket),
		Key:    aws.String(key),
	}
	return client.svc().WaitUntilObjectNotExists(headObjectInput)
}
