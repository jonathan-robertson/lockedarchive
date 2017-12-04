package service

import "github.com/jonathan-robertson/lockedarchive/secure"

// AS3Location represents a remote storage location in Amazon S3
type AS3Location struct {
	Bucket    string `json:"bucket,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
}

// getBucket decrypts the loaded Bucket
func (as3 AS3Location) getBucket() (*secure.SecretContainer, error) {
	return secure.DecryptWithSaltFromStringToSecret(passphrase, as3.Bucket)
}

// getAccessKey decrypts the loaded AccessKey
func (as3 AS3Location) getAccessKey() (*secure.SecretContainer, error) {
	return secure.DecryptWithSaltFromStringToSecret(passphrase, as3.AccessKey)
}

// getSecretKey decrypts the loaded SecretKey
func (as3 AS3Location) getSecretKey() (*secure.SecretContainer, error) {
	return secure.DecryptWithSaltFromStringToSecret(passphrase, as3.SecretKey)
}
