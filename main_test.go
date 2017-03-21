package main

import (
	"testing"

	"os"

	"time"

	"github.com/blevesearch/bleve"
	"github.com/puddingfactory/textify"
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
	indexName = "example.bleve"
	filename  = "testing/test.pdf"
)

var (
	exampleIndex bleve.Index
)

func getContents() (fc FileContents, err error) {
	body, err := textify.PDF(filename, "\n")
	if err != nil {
		return
	}

	info, err := os.Stat(filename)
	if err != nil {
		return
	}

	fc = FileContents{
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
		Name:    info.Name(),
		Size:    info.Size(),
		Body:    body,
	}
	return
}

// setup creates index and loads it with default values
func setup() error {
	contents, err := getContents()
	if err != nil {
		return err
	}
	contents.ID = "thisIsTheId"

	mapping := bleve.NewIndexMapping()
	if exampleIndex, err = bleve.New(indexName, mapping); err != nil {
		if exampleIndex, err = bleve.Open(indexName); err != nil {
			return err
		}
	}

	exampleIndex.Index(contents.ID, contents)
	return nil
}

// teardown removes index and contents
func teardown(t *testing.T) {
	if err := exampleIndex.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(indexName); err != nil {
		t.Fatal(err)
	}
}

func TestMain(t *testing.T) {
	defer teardown(t)
	if err := setup(); err != nil {
		t.Fatal(err)
	}

	t.Run("Index", func(t *testing.T) { IndexSearch(t) })
	t.Run("Highlight", func(t *testing.T) { HighlightMatches(t) })
}

func IndexSearch(t *testing.T) {
	query := bleve.NewQueryStringQuery("contact")
	searchRequest := bleve.NewSearchRequest(query)

	searchResult, err := exampleIndex.Search(searchRequest)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("searchResult: %+v\n", searchResult)
}

func HighlightMatches(t *testing.T) {
	query := bleve.NewMatchQuery("contact")
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Highlight = bleve.NewHighlight()
	searchResults, err := exampleIndex.Search(searchRequest)
	if err != nil {
		t.Fatal(err)
	}

	for i, hit := range searchResults.Hits {
		t.Logf("HIT %d: %+v\n", i, hit.Fragments["Body"])
	}
	// Output:
	// great <mark>nameless</mark> one
}
