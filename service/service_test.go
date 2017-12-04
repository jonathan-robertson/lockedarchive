package service_test

import (
	"encoding/json"
	"testing"

	"github.com/jonathan-robertson/lockedarchive/service"
)

/*
TDD Test Plan

	Configuration
		load configuration
		create archive
		add cloud storage
		remove cloud storage
		remove archive
		save configuration

	Actions
		put
		get
		getInfo
		delete
		update
		search
*/

const (
	testFilename = "testSettings.config"
)

func TestConfig(t *testing.T) {
	// using a new byte slice each time is necessary because the slice is
	// wiped when converted to a passphrase container / key
	expectActivationSuccess(t, makeGoodPassphrase())
	t.Log("test Config file Loaded")

	location := service.AS3Location{
		Bucket:    "testBucket",
		AccessKey: "testAccessKey",
		SecretKey: "testSecretKey",
	}
	locationByteSlice, err := json.Marshal(location)
	if err != nil {
		t.Fatal(err)
	}

	kc, err := service.CreateArchive("test")
	if err != nil {
		t.Fatal(err)
	}
	defer kc.Destroy()
	t.Log("archive added and config saved")

	if err := service.AddLocations("test", locationByteSlice); err != nil {
		t.Fatal(err)
	}
	t.Log("location added and config saved")

	expectActivationFailure(t, makeBadPassphrase())
	expectActivationSuccess(t, makeGoodPassphrase())

	if err := service.RemoveConfiguration(); err != nil {
		t.Fatal(err)
	}
	t.Log("test Config file successfully deleted")
}

func makeGoodPassphrase() []byte {
	return []byte("correct")
}

func makeBadPassphrase() []byte {
	return []byte("incorrect")
}

func expectActivationSuccess(t *testing.T, passphrase []byte) {
	if err := attemptActivation(passphrase); err != nil {
		t.Fatal(err)
	}
	t.Log("activation succeeded, as expected")
}

func expectActivationFailure(t *testing.T, passphrase []byte) {
	if err := attemptActivation(passphrase); err == nil {
		t.Fatal(err)
	}
	t.Log("activation failed, as expected")
}

func attemptActivation(passphrase []byte) error {
	return service.ActivateService(passphrase, testFilename)
}
