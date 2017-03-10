package main

import (
	"fmt"
	"testing"

	"rsc.io/pdf"
)

const (
	filename          = "testing/test.pdf"
	threshold float64 = .8
)

func TestRun(t *testing.T) {
	r, err := pdf.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= r.NumPage(); i++ {
		var prev pdf.Text
		for _, cur := range r.Page(i).Content().Text {

			switch true {
			case formatChanged(prev, cur):
			case lineChanged(prev, cur):
			case spacingChanged(prev, cur):
				fmt.Print("\n")
			}

			fmt.Print(cur.S)
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
	fmt.Println()
}

func formatChanged(prev, cur pdf.Text) bool {
	return prev.Font != cur.Font || prev.FontSize != cur.FontSize
}

func lineChanged(prev, cur pdf.Text) bool {
	return prev.Y != cur.Y
}

func spacingChanged(prev, cur pdf.Text) bool {
	// todo
	return false
}
