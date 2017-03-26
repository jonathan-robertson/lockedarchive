package main

import (
	"testing"

	"os"

	"time"

	"github.com/blevesearch/bleve"
)

type FileContents struct {
	ID      string
	Name    string
	Body    string
	ModTime time.Time
	IsDir   bool
	Size    int64
}

const (
	indexName = "test.bleve"

	testBody = `you never get enough of them
there is nothing you could say to get rid of them
fish is great, it is lovely
it is the answer to our prayers
this is none other like you
nobody compares to you
oh how I love you
fish`
)

var (
	testIndex bleve.Index
)

// setup creates index and loads it with default values
func setup(t *testing.T) {
	var err error
	mapping := bleve.NewIndexMapping()
	if testIndex, err = bleve.New(indexName, mapping); err != nil {
		if testIndex, err = bleve.Open(indexName); err != nil {
			t.Fatal(err)
		}
	}
}

// teardown removes index and contents
func teardown(t *testing.T) {
	if err := testIndex.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(indexName); err != nil {
		t.Fatal(err)
	}
}

func TestMain(t *testing.T) {
	defer teardown(t)
	setup(t)

	t.Run("IndexText", func(t *testing.T) { IndexText(t) })
	t.Run("SearchIndex", func(t *testing.T) { SearchIndex(t) })
	t.Run("Highlight", func(t *testing.T) { HighlightMatches(t) })
}

func IndexText(t *testing.T) {
	if err := testIndex.Index("example", FileContents{Body: testBody}); err != nil {
		t.Fatal(err)
	}
}

func SearchIndex(t *testing.T) {
	query := bleve.NewQueryStringQuery("fish")
	searchRequest := bleve.NewSearchRequest(query)

	searchResult, err := testIndex.Search(searchRequest)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("searchResult: %+v\n", searchResult)
}

func HighlightMatches(t *testing.T) {
	query := bleve.NewMatchQuery("fish")
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Highlight = bleve.NewHighlight()
	searchResults, err := testIndex.Search(searchRequest)
	if err != nil {
		t.Fatal(err)
	}

	for _, hit := range searchResults.Hits {
		t.Logf(
			"HIT: %d, ID: %s, Score: %f\n%+v\n",
			hit.HitNumber, hit.ID, hit.Score, hit.Fragments,
		)
	}
	// Output:
	// HIT: 1, ID: example, Score: 0.079229
	// map[Body:[<mark>fish</mark> is great, it is lovely
	// you never get enough of them
	// there is nothing you could say to get rid of them
	// it is the answer to our prayers
	// this is none other like you
	// nobody compares to you
	// oh how I l…]]
}
