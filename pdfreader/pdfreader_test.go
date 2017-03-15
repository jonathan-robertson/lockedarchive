package pdfreader

import (
	"fmt"
	"strings"
	"testing"
)

var (
	filenames = []string{
		"testing/test.pdf",
	}
)

func TestExtraction(t *testing.T) {
	for _, filename := range filenames {
		t.Run(fmt.Sprintf("%12s", filename), func(t *testing.T) { countExtracted(t, filename) })
	}
}

func countExtracted(t *testing.T, filename string) {
	words, err := ExtractWords(filename)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%10d words detected\n", len(words))
}

// Unused; leaving in as example for user
func printWords(t *testing.T) {
	words, err := ExtractWords(filenames[0])
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("\n%s\n", strings.Join(words, " "))
}
