package clob // Cloud Object
import "io"

// Entry represents a cloud object
type Entry struct {
	// Note from Amazon on naming:
	// Alphanumeric characters [0-9a-zA-Z]
	// Special characters !, -, _, ., *, ', (, and )
	// TODO: more notes are on that page... read them more and consider them

	// REVIEW: research and consider search values...
	// Tags could be recorded as metadata, comma-delimited... on Unmarshal, could have tagMap (map[string][]string == map[tag][]GUIDs)

	// TODO: 32 (?) chars of hex (?), incremented and inverted so that 00...01 becomes 10...00
	// NOTE: This is all you'll see in S3
	ID string

	// TODO: Store this value in metadata? Or would it make more sense to store it as a prefix so we can do a lookup to get what's immediately inside a dir.
	// NOTE: Is not nested
	ParentID string

	// TODO: Adhere Windows' to standard... or S3's?
	// NOTE: Also in metadata..?
	// Name string

	// REVIEW: maybe should offload this to local FS instead (cache).
	// REVIEW: maybe pass around an io.Reader instead
	Body io.Reader

	// Size in bytes of data
	Size int64

	// REVIEW: Does a rune actually work here? Would take less steps to use string instead.
	EntryType rune

	// NOTE: The PUT request header is limited to 8 KB in size. Within the PUT request header, the user-defined metadata is limited to 2 KB in size. The size of user-defined metadata is measured by taking the sum of the number of bytes in the UTF-8 encoding of each key and value
	Metadata map[string]string
}

func (e Entry) Key() string {
	return e.ParentID + "/" + e.ID
}

// func (e *Entry) EncryptData() (err error) {
// 	e.Data, err = crypt.Encrypt(e.Data)
// 	return
// }

// func (e *Entry) EncryptName() (err error) {
// 	e.Name, err = crypt.EncryptStringToHexString(e.Name)
// 	return
// }

// func (e *Entry) DecryptData() (err error) {
// 	e.Data, err = crypt.Decrypt(e.Data)
// 	return
// }

// func (e *Entry) DecryptName() (err error) {
// 	e.Name, err = crypt.DecryptHexStringToString(e.Name)
// 	return
// }
