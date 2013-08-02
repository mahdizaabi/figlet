package main

import (
	"strings"
	//"fmt"
)

// smush modes
const (
	SMEqual = 1
	SMLowLine = 2
	SMHierarchy = 4
	SMPair = 8
	SMBigX = 16
	SMHardBlank = 32
	SMKern = 64
	SMSmush = 128
)

// Given 2 characters, attempts to smush them into 1, according to
// smushmode.  Returns smushed character or '\0' if no smushing can be
// done.

// smushmode values are sum of following (all values smush blanks):
// 1: Smush equal chars (not hardblanks)
// 2: Smush '_' with any char in hierarchy below
// 4: hierarchy: "|", "/\", "[]", "{}", "()", "<>"
//    Each class in hier. can be replaced by later class.
// 8: [ + ] -> |, { + } -> |, ( + ) -> |
// 16: / + \ -> X, > + < -> X (only in that order)
// 32: hardblank + hardblank -> hardblank
func smushem(lch rune, rch rune, s settings) rune {
	if lch == ' ' { return rch }
	if rch == ' ' { return lch }

	if s.smushmode & SMSmush == 0 { // smush not enabled
		return 0
	}

	if s.smushmode & SMKern == 0 { // smush but not kern
		// This is smushing by universal overlapping

		// ensure overlapping preference to visible chars (spaces handled already)
		if lch == s.hardblank { return rch }
		if rch == s.hardblank { return lch }

		// ensure dominant char overlaps, depending on right-to-left parameter
		if s.rtol { return lch }
		return rch
	}

	if s.smushmode & SMHardBlank == SMHardBlank {
		if lch == s.hardblank && rch == s.hardblank { return s.hardblank }
	}

	if lch == s.hardblank || rch == s.hardblank { return 0 }

	if s.smushmode & SMEqual == SMEqual {
		if lch == rch { return lch }
	}

	if s.smushmode & SMLowLine == SMLowLine {
		if lch == '_' && strings.ContainsRune("|/\\[]{}()<>", rch) { return rch }
		if rch == '_' && strings.ContainsRune("|/\\[]{}()<>", lch) { return lch }
	}

	if s.smushmode & SMHierarchy == SMHierarchy {
		hrchy := []string { "|", "/\\", "[]", "{}", "()", "<>" } // low -> high precedence
		inHrchy := func(low rune, high rune, i int) bool {
			return strings.ContainsRune(hrchy[i], low) && strings.ContainsRune(strings.Join(hrchy[i+1:], ""), high)
		}
		for i, _ := range hrchy {
			if inHrchy(lch, rch, i) { return rch }
			if inHrchy(rch, lch, i) { return lch }
		}
	}

	if s.smushmode & SMPair == SMPair {
		if lch=='[' && rch==']' { return '|' }
		if rch=='[' && lch==']' { return '|' }
		if lch=='{' && rch=='}' { return '|' }
		if rch=='{' && lch=='}' { return '|' }
		if lch=='(' && rch==')' { return '|' }
		if rch=='(' && lch==')' { return '|' }
	}

	if s.smushmode & SMBigX == SMBigX {
		if lch == '/' && rch == '\\' { return '|' }
		if lch == '\\' && rch == '/' { return 'Y' }
		if lch == '>' && rch == '<' { return 'X' }
	}
	return 0
}

// smushamt returns the maximum amount that the character can be smushed
// into the line.
func smushamt(char *figText, line *figText, s settings) int {
	if s.smushmode & (SMSmush | SMKern) == 0 {
		return 0;
  	}

  	empty := func (ch rune) bool {
		return ch == 0 || ch == ' '
	}

	maxsmush := char.width()
	for row := 0; row < char.height(); row++ {
		var left, right []rune
		if s.rtol {
			left, right = (*char).art[row], (*line).art[row]
		} else {
			left, right = (*line).art[row], (*char).art[row]
		}

		// find number of empty chars for left and right
		var i, j int
		for i = 0; i < len(left) && empty(left[len(left) - 1 - i]); i++ { }
		for j = 0; j < len(right) && empty(right[j]); j++ { }

		// the amount of smushing possible just by removing empty spaces
		rowsmush := j + i

		if i < len(left) && j < len(right) {
			// see if we can smush it even further
			lch := left[len(left) - 1 - i]
			rch := right[j]
			if !empty(lch) && !empty(rch) {
				if smushem(lch, rch, s) != 0 { rowsmush++ }
			}
		}

		//fmt.Printf("i: %v, j: %v, rowsmush: %v\n", i, j, rowsmush)

		if rowsmush < maxsmush { maxsmush = rowsmush }
	}

	return maxsmush;
}

type settings struct {
	smushmode int
	hardblank rune
	rtol bool
}

// Adds the given character onto the end of the given line.
func addChar(char *figText, line *figText, s settings) {
	smushamount := smushamt(char, line, s)

	linelen := line.width()

	for row := 0; row < char.height(); row++ {
		if s.rtol { panic ("right-to-left not implemented") }
		for k := 0; k < smushamount; k++ {
			column := linelen - 1

			rch := (*char).art[row][k]
			var smushed rune
			if column < 0 {
				column = 0
				smushed = rch
			} else {
				lch := (*line).art[row][column]
				smushed = smushem(lch, rch, s)
			}
			
			(*line).art[row] = append((*line).art[row][:column], smushed)
		}
		(*line).art[row] = append((*line).art[row], (*char).art[row][smushamount:]...)
	}
}

// gets the font entry for the given character, or the 'missing'
// character if the font doesn't contain this character
func getChar(c rune, f font) *figText {
	 l, ok := f.chars[c]
	 if !ok {
		l = f.chars[0]
	 }
	 return &figText { text: string(c), art: l }
}

func getWord(w string, f font, s settings) *figText {
	word := newFigText(f.header.charheight)
	(*word).text = w
	for _, c := range w {
		c := getChar(c, f)
		addChar(c, word, s)
	}

	return word
}

func getWords(msg string, f font, s settings) []figText {
	words := make([]figText, 0)
	for _, word := range strings.Split(msg, " ") {
		words = append(words, *getWord(word, f, s))
	}
	return words
}

func getLines(msg string, f font, maxwidth int, s settings) []figText {
	lines := make([]figText, 1)
	words := getWords(msg, f, s)

	// empty first line
	lines[0] = *newFigText(f.header.charheight)

	i := 0
	for _, word := range words {
		if lines[i].width() + word.width() > maxwidth { // need to wrap
			lines = append(lines, figText { art: make([][]rune, f.header.charheight) })

			if word.width() > maxwidth {
				a, b := word.splitAt(maxwidth - lines[i].width() - 1)

				// code dupe
				for j, wordline := range a.art {
					lines[i].art[j] = append(lines[i].art[j], wordline...)
				}
				word = *b
			}

			i++
		}

		for j, wordline := range word.art {
			lines[i].art[j] = append(lines[i].art[j], wordline...)
		}
	}

	return lines
}