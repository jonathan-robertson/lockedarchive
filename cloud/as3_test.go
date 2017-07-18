package cloud

import "testing"

func TestAS3(t *testing.T) {
	err := AS3Client("lockedarchive-test", "us-east-1").CreateArchive()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success: created archive with AS3")
}
