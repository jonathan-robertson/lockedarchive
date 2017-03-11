package main

import (
	"fmt"
	"testing"

	"strings"

	"rsc.io/pdf"
)

const (
	filename          = "testing/test.pdf"
	threshold float64 = 0.01 // min/max diff required for space detection
)

var (
	words []string
)

func TestWordParser(t *testing.T) {
	r, err := pdf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	// Print pages
	for i := 1; i <= r.NumPage(); i++ {
		words := parseTextStream(r.Page(i).Content().Text)
		fmt.Printf("\nPage %d\n%s\n", i, strings.Join(words, " "))
	}
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

func streamToString(stream []pdf.Text) (text string) {
	for _, letter := range stream {
		text += letter.S
	}
	return
}

func TestRelationalSpacing(t *testing.T) {
	r, err := pdf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	var letters []pdf.Text
	for i := 1; i <= r.NumPage(); i++ {
		for _, letter := range r.Page(i).Content().Text {

			// Check for word ending
			if len(letters) > 0 {
				prev := letters[len(letters)-1]
				if formatChanged(prev, letter) || lineChanged(prev, letter) {
					outputStats(letters)
					letters = []pdf.Text{} // reset to only hold this letter
				}
			}

			// Add letter
			letters = append(letters, letter)
		}
		outputStats(letters) // print final word's stats
	}
}

func outputStats(letters []pdf.Text) {
	var word string
	for _, letter := range letters {
		word += letter.S
	}
	// fmt.Printf("\n%s\n", word)
	fmt.Printf("\n====================\n%s\n", word)

	tallyDist(letters)

	/*
		Given a[x]b where x equals the empty space between the end of a and beginning of b, knowing whether a and b are part of the same word is not possible.

		However; adding c allows us to know this.

		For example, a[x]b[y]c tells us that a doesn't belong to the word that b or c belongs to... And we don't know if b and c belong to the same word.

		So there's this kind of validation phase when moving through the letters.
	*/

	for i, letter := range letters {
		fmt.Printf("%4d. %s\n", i, letter.S)
		if i < len(letters)-1 { // for every instance except for last
			next := letters[i+1]
			dist := next.X - letter.W - letter.X
			fmt.Printf("\t%20.10f\n", dist)
		}
	}
}

func tallyDist(letters []pdf.Text) {
	var (
		prev, cur pdf.Text
		distances []float64
	)

	for _, letter := range letters {

		// Rotate values forward
		prev = cur
		cur = letter

		// Rotate in another value if prev doesn't have one yet
		if prev.S == "" {
			continue
		}

		// Collect distance
		distances = append(distances, cur.X-prev.W-prev.X)
	}

	avg, isMultiword := minMaxAvg(distances)
	fmt.Printf("\tmulti-word: %t\t%10.6f avg\n", isMultiword, avg)

	// Tally up distances
	distMap := make(map[float64]int)
	for _, d := range distances {
		distMap[d]++
	}

	// Print dist stats
	for key, val := range distMap {
		fmt.Printf("%14.5f: %3d space: %t\n", key, val, isMultiword && key > avg)
	}
	fmt.Println() // add blank line
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

func TestRun(t *testing.T) {
	r, err := pdf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= r.NumPage(); i++ {
		var prev pdf.Text
		var word string
		for _, cur := range r.Page(i).Content().Text {
			if formatChanged(prev, cur) || lineChanged(prev, cur) || spacingChanged(prev, cur) {

				if !formatChanged(prev, cur) && !lineChanged(prev, cur) {
					dist := cur.X - prev.X - prev.W
					if dist > threshold {
						fmt.Printf("pw: %7.4f f: %10s fs: %4.1f dist: %7.4f word: ", prev.W, prev.Font, prev.FontSize, dist)
					}
				}

				fmt.Println(word)
				word = cur.S // reset word, starting with this char
				prev = cur
				continue // skip since space detected
			}
			word += cur.S // concat this char to growing word
			prev = cur

			// var spacing float64
			// var newline bool
			// if prev.Y == text.Y {
			// 	spacing = text.X - (prev.X + prev.W)
			// } else {
			// 	newline = true
			// }

			// // TODO:detect and handle hyphenation.
			// // If newline, does prev.S equal "-"?
			// // If so, remove hyphen (?) and consider cur to be part of the ongoing word

			// // TODO: base threshold on font size?
			// if newline {
			// 	fmt.Printf("\n\t[%+v|%+v]:%f newline\n", prev, text, spacing)
			// }

			// if spacing > threshold {
			// 	fmt.Printf("\n\t[%+v|%+v]:%f\n", prev, text, spacing)
			// }

			// fmt.Print(text.S)
			// // fmt.Printf("%t\t%f\t%s\t%+v\n", newline, spacing, text.S, text)
		}
	}
	// fmt.Println()
	// printAnalysis()
}

func formatChanged(prev, cur pdf.Text) bool {
	return prev.Font != cur.Font || prev.FontSize != cur.FontSize
}

func lineChanged(prev, cur pdf.Text) bool {
	return prev.Y != cur.Y
}

func spacingChanged(prev, cur pdf.Text) bool {
	dist := cur.X - prev.X - prev.W
	return dist > threshold
}
