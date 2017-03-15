// Package pdfreader extracts useable text from PDF files.
package pdfreader

import "rsc.io/pdf"

const (
	threshold float64 = 0.01 // min/max diff required for space detection
)

// ExtractWords pulls a slice of words from a given PDF file.
// Join slice with space to get readable output.
func ExtractWords(filename string) (words []string, err error) {
	r, err := pdf.Open(filename)
	if err != nil {
		return
	}

	// Print pages
	for i := 1; i <= r.NumPage(); i++ {
		words = append(words, parseTextStream(r.Page(i).Content().Text)...)
	}

	return
}

func parseTextStream(stream []pdf.Text) (words []string) {
	for _, phrase := range parsePhrases(stream) {
		var (
			prev, cur pdf.Text
			distances []float64
		)

		for _, letter := range phrase {
			prev = cur
			cur = letter
			if prev.S == "" {
				continue
			}
			distances = append(distances, cur.X-prev.W-prev.X)
		}

		if avg, isMultiword := minMaxAvg(distances); isMultiword {
			var word string
			for i := 0; i < len(phrase); i++ {
				word += phrase[i].S
				if i < len(distances) && distances[i] > avg {
					words = append(words, word)
					word = ""
				}
			}
			words = append(words, word) // append final word
		} else {
			var word string
			for _, letter := range phrase {
				word += letter.S
			}
			words = append(words, word) // append final word
		}
	}
	return
}

func parsePhrases(stream []pdf.Text) (phrases [][]pdf.Text) {
	var (
		prev, cur pdf.Text
		phrase    []pdf.Text
	)

	for _, letter := range stream {
		prev = cur
		cur = letter
		if prev.S == "" {
			phrase = append(phrase, cur)
			continue
		}

		// Start of new phrase?
		if formatChanged(prev, cur) || lineChanged(prev, cur) {
			phrases = append(phrases, phrase)
			phrase = []pdf.Text{} // reset phrase
		}

		phrase = append(phrase, cur)
	}
	phrases = append(phrases, phrase) // append final
	return
}

func minMaxAvg(vals []float64) (avg float64, isMultiword bool) {
	if len(vals) == 0 {
		return
	}
	min := vals[0]
	max := vals[0]

	for _, v := range vals {
		switch {
		case v < min:
			min = v
		case v > max:
			max = v
		}
	}

	if max-min > threshold {
		isMultiword = true
		avg = (min + max) / 2
	}
	return
}

func formatChanged(prev, cur pdf.Text) bool {
	return prev.Font != cur.Font || prev.FontSize != cur.FontSize
}

func lineChanged(prev, cur pdf.Text) bool {
	return prev.Y != cur.Y
}
