// Package service responsible for providing async commands for REST API to call
package service

import (
	"github.com/shibukawa/configdir"

	"github.com/jonathan-robertson/lockedarchive/secure"
)

const configFilename = "lockedArchive.config"

var config configdir.Config

type configContents struct {
	Caches map[string][]map[string]string `json:"caches"` // name: slices of maps
}

func init() {
	// TODO: load properties/preferences

}

func loadPreferences(key secure.Key) error {

	return nil
}
