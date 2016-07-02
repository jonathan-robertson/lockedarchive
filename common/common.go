package common

import "sync"

type Cabinet struct {
	Name string // aws bucket
	sync.RWMutex
	FileMap map[string]File
}

type File struct {

	// Note from Amazon on naming:
	// Alphanumeric characters [0-9a-zA-Z]
	// Special characters !, -, _, ., *, ', (, and )
	// TODO: more notes are on that page... read them more and consider them

	// REVIEW: research and consider search values...
	// Tags could be recorded as metadata, comma-delimited... on Unmarshal, could have tagMap (map[string][]string == map[tag][]GUIDs)

	// TODO: 32 (?) chars of hex (?), incremented and inverted so that 00...01 becomes 10...00
	// NOTE: This is all you'll see in S3
	ID string

	// TODO: Adhere Windows' to standard... or S3's?
	// NOTE: Also in metadata..?
	Name string

	// TODO: Store this value in metadata
	// TODO: Verify how many characters can be in a metadata value
	// NOTE: can be "nested" - /Vital/Jonathan.
	Folder string

	// TODO: Build fetch func for downloading object's bytes and storing to Data if requested (don't 'load ahead')
	// NOTE: data that can be streamed to a file on system if download is opted for
	Data []byte
}

// type Folder struct {
//     Parent string
//     Name string
// }
