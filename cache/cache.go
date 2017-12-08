// Package cache is responsible for managing the local cache
package cache

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/shibukawa/configdir"

	"github.com/jonathan-robertson/lockedarchive/cloud"
	"github.com/jonathan-robertson/lockedarchive/secure"
	"github.com/jonathan-robertson/lockedarchive/stream"
)

const (
	vendorName = "com.lockedarchive"
	appName    = "lockedarchive"
)

var (
	cacheConfig *configdir.Config

	errFailedToWrite = errors.New("failed to write file to cache")
)

func init() {
	configDirs := configdir.New(vendorName, appName)
	cacheConfig = configDirs.QueryCacheFolder()
}

// TODO: get:			download from online storage
// TODO: free:		delete from cache (not online storage)
// TODO: remove:	delete from cache (if exists) and online storage
// ---------------------------------------------------------------
// TODO: read:		decrypt for ram or local filesystem

// Write analyzes, encrypts, and compresses a new file into the cache
// NOTE: this will overwrite the data currently existing in cache for this entity
func Write(ctx context.Context, pc *secure.PassphraseContainer, parentID string, path string) error {
	srcFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	kc, err := secure.GenerateKeyContainer()
	if err != nil {
		return err
	}
	keyStr, err := secure.EncryptWithSaltToString(pc, kc.Buffer())
	if err != nil {
		return err
	}

	entry, err := fileToEntry(ctx, srcFile, parentID, keyStr)
	if err != nil {
		return err
	}

	cacheFile, err := cacheConfig.Create(entry.ID)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	// streamToFile is expected to close srcFile and cacheFile
	if err := streamToFile(ctx, srcFile, cacheFile, kc); err != nil {
		return err
	}

	// TODO: add metadata to bolt

	// TODO: produce checksum of file data

	// TODO: add entry to each storage provider for upload

	return nil
}

func fileToEntry(ctx context.Context, file *os.File, parentID, keyStr string) (*cloud.Entry, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if stream.IsTooLargeToChunk(info.Size()) {
		return nil, stream.ErrEncryptSize
	}

	entry := &cloud.Entry{
		ID:           generateID(),
		Key:          keyStr,
		ParentID:     parentID,
		Name:         info.Name(),
		IsDir:        info.IsDir(),
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Mode:         info.Mode(),
	}
	return entry, nil
}

// streamToFile compresses and encrypts contents as a stream from src to dst then closes src and dst once done
func streamToFile(ctx context.Context, src, dst *os.File, kc *secure.KeyContainer) error {
	streamCtx, cancelCtx := context.WithCancel(ctx)

	pcr, pcw := io.Pipe()
	errChan := make(chan error, 5)
	go func() {
		if _, err := stream.Compress(src, pcw); err != nil {
			errChan <- err
			cancelCtx()
		}
		if err := pcw.Close(); err != nil {
			errChan <- err
		}
	}()

	per, pew := io.Pipe()
	go func() {
		if _, err := stream.Encrypt(streamCtx, kc, pcr, pew); err != nil {
			errChan <- err
			cancelCtx()
		}
		if err := pew.Close(); err != nil {
			errChan <- err
		}
	}()

	if _, err := io.Copy(dst, per); err != nil {
		errChan <- err
	}
	close(errChan)

	var err error // starting as nil
	for errFromChan := range errChan {
		err = errors.New(err.Error() + "; " + errFromChan.Error())
	}
	if err != nil {
		return err
	}

	if err := dst.Sync(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}

	return src.Close()
}

// generateID returns a randomly generated ID for use in a new Entry
func generateID() string {
	return "temp" // TODO: actually generate something
}

// // Download fetches Entry from cloud provider(s) for local cache
// // TODO
// func Download(id string) error {
// 	return nil
// }

// // Put adds a file to the cache without any modifications
// // This is best used with data received from cloud storage
// // TODO: update to use cache path
// func Put(path string, rc io.ReadCloser) (err error) {
// 	file, err := os.Create(path)
// 	if err != nil {
// 		return
// 	}
// 	defer file.Close() // backup in case we don't reach file.Sync

// 	if _, err = io.Copy(file, rc); err != nil {
// 		return err
// 	}

// 	if err := rc.Close(); err != nil {
// 		return err
// 	}

// 	return file.Sync()
// }

// // Get returns readCloser to a cached file; caller responsible for closing
// // This is best used for providing cloud storage the bytes to transmit
// // TODO: update to use cache path
// func Get(path string) (*os.File, error) {
// 	return os.Open(path)
// }

// // seal compresses and encrypts a file at provided path, writing it to the cache
// // TODO: Do not receive key; already have it
// func seal(path string, kc *secure.KeyContainer) error {
// 	src, err := os.Open(path)
// 	if err != nil {
// 		return err
// 	}
// 	defer src.Close()

// 	dst, err := os.Create(src.Name() + ".la")
// 	if err != nil {
// 		return err
// 	}
// 	defer dst.Close()

// 	pr, pw := io.Pipe()
// 	defer pw.Close()
// 	ctx, cancel := context.WithCancel(context.Background())

// 	// Compress data
// 	var compressionErr error
// 	go func() {
// 		if _, compressionErr = stream.Compress(src, pw); compressionErr != nil {
// 			cancel()
// 		}
// 		if compressionErr = pw.Close(); compressionErr != nil {
// 			cancel()
// 		}
// 	}()

// 	// Encrypt data
// 	if _, err = stream.Encrypt(ctx, kc, pr, dst); err != nil {
// 		if err == context.Canceled {
// 			return compressionErr
// 		}
// 		return err
// 	}

// 	return dst.Sync()
// }

// // unseal decrypts and decompresses a file at provided path
// // TODO: update to use name/key
// func unseal(path string, kc *secure.KeyContainer) error {
// 	src, err := os.Open(path)
// 	if err != nil {
// 		return err
// 	}
// 	defer src.Close()

// 	dst, err := os.Create(strings.TrimSuffix(src.Name(), ".la"))
// 	if err != nil {
// 		return err
// 	}
// 	defer dst.Close() // backup in case we don't reach dst.Sync

// 	pr, pw := io.Pipe()
// 	defer pw.Close()
// 	ctx, cancel := context.WithCancel(context.Background())

// 	// Decrypt data
// 	var decryptionErr error
// 	go func() {
// 		if _, decryptionErr = stream.Decrypt(ctx, kc, src, pw); decryptionErr != nil {
// 			cancel() // TODO: THIS DOESN'T DO REALLY ANYTHING
// 		}
// 		if decryptionErr = pw.Close(); decryptionErr != nil {
// 			cancel() // TODO: THIS DOESN'T DO REALLY ANYTHING
// 		}
// 	}()

// 	// Decompress data
// 	if _, err := stream.Decompress(pr, dst); err != nil {
// 		return err
// 	}

// 	return dst.Sync()
// }
