// Package service responsible for providing async commands for REST API to call
package service

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"

	"github.com/shibukawa/configdir"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

// TODO: turn these into var and allow override for testing purposes
const (
	vendorName = "com.lockedarchive"
	appName    = "lockedarchive"
)

var (
	config     *Configuration
	passphrase *secure.PassphraseContainer

	errArchiveAlreadyExits  = errors.New("Archive already exists")
	errArchiveDoesNotExit   = errors.New("Archive does not exist")
	errInvalidLocation      = errors.New("Invalid online storage location")
	errLocationAlreadyInUse = errors.New("Online storage location already in use")
	errPassphraseNotSet     = errors.New("passphrase not set")
)

// AS3Location represents a remote storage location in Amazon S3
type AS3Location struct {
	Bucket    string `json:"bucket,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
}

// Archive represents sets of locations meant to store the same dataset
type Archive struct {
	AmazonS3 map[string]AS3Location `json:"amazon_s3,omitempty"`
}

// Configuration represents the structure of our config file
type Configuration struct {
	Filename string
	Archives map[string]Archive `json:"archives,omitempty"`
}

// ActivateService initiates necessary steps for service to run
func ActivateService(pass []byte, filename string) (err error) {
	passphrase, err = secure.ProtectPassphrase(pass)
	if err != nil {
		return
	}
	// TODO: planning to use "settings.config" in prod
	return loadConfig(filename)
}

// CreateArchive adds an archive to the config file and saves config
func CreateArchive(archiveName string, locationData ...[]byte) error {
	if _, exists := config.Archives[archiveName]; exists {
		return errArchiveAlreadyExits
	}

	config.Archives[archiveName] = Archive{
		AmazonS3: make(map[string]AS3Location),
	}

	if err := saveConfig(); err != nil {
		return err
	}

	return AddLocations(archiveName, locationData...)
}

// AddLocations adds a remote location to an existing archive
func AddLocations(archiveName string, data ...[]byte) error {
	archive, exists := config.Archives[archiveName]
	if !exists {
		return errArchiveDoesNotExit
	}

	for _, location := range data {
		// TODO: check for other types here instead when other types added
		s3, err := bytesToAS3Location(location)
		if err != nil {
			return errInvalidLocation
		}

		archive.AmazonS3[s3.Bucket] = s3
		config.Archives[archiveName] = archive

		if err := saveConfig(); err != nil {
			return err
		}
	}

	return nil
}

// RemoveConfiguration removes the config file from the file system
func RemoveConfiguration() error {
	return deleteConfig()
}

func bytesToAS3Location(data []byte) (s3Loc AS3Location, err error) {
	err = json.Unmarshal(data, &s3Loc)
	return
}

func loadConfig(filename string) error {
	configDirs := configdir.New(vendorName, appName)

	path, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	configDirs.LocalPath = path

	folder := configDirs.QueryFolderContainsFile(filename)

	// Handle if the config file doesn't exist yet in filesystem
	if folder == nil {
		config = &Configuration{
			Filename: filename,
			Archives: make(map[string]Archive),
		}
		return nil
	}

	data, err := folder.ReadFile(config.Filename)
	if err != nil {
		return err
	}

	decryptedData, err := secure.DecryptWithSalt(passphrase, data)
	if err != nil {
		return err
	}

	return json.Unmarshal(decryptedData, &config)
}

func saveConfig() error {
	configDirs := configdir.New(vendorName, appName)
	folders := configDirs.QueryFolders(configdir.Global)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	nonce, err := secure.GenerateNonce()
	if err != nil {
		return err
	}

	// TODO: wipe salt, encyptedData, and combo once done?
	contents, err := secure.EncryptWithSaltAndWipe(passphrase, nonce, data)
	if err != nil {
		return err
	}

	return folders[0].WriteFile(config.Filename, contents)
}

func deleteConfig() error {
	configDirs := configdir.New(vendorName, appName)
	folder := configDirs.QueryFolderContainsFile(config.Filename)
	if folder == nil {
		return nil
	}

	return os.Remove(path.Join(folder.Path, config.Filename))
}
