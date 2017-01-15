package clob // Cloud Object
import (
	"io"
	"time"
)

// Entry represents a cloud object
type Entry struct {
	Key          string
	ParentKey    string
	Name         string
	Type         rune
	Size         int64
	LastModified time.Time
	Body         io.ReadCloser
	// Tags ??? Tags could be recorded as metadata, comma-delimited... on Unmarshal, could have tagMap (map[string][]string == map[tag][]GUIDs)
}
