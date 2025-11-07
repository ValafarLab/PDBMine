package main

import (
	"testing"
)

func TestTranslateToOneLetterCode(t *testing.T) {
	threeLetter := []string{"VAL", "TRP", "SER", "PHE", "LYS", "ILE", "GLY", "GLN", "CYS", "ASP", "ARG", "AXB", "ALA", "ASN", "ASX", "GLU", "GLX", "HIS", "LEU", "MET", "PRO", "THR", "TYR"}

	oneLetterExpected := []string{"V", "W", "S", "F", "K", "I", "G", "Q", "C", "D", "R", "X", "A", "N", "B", "E", "Z", "H", "L", "M", "P", "T", "Y"}

	oneLetterActual := TranslateToOneLetterCode(threeLetter)

	for i, code := range oneLetterExpected {
		if code != oneLetterActual[i] {
			t.Errorf("Error translating %s. Expected %s. Actual %s.", threeLetter[i], oneLetterExpected[i], oneLetterActual[i])
		}
	}
}

func TestFindIntersection(t *testing.T) {
	setOne := []uint32{2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30}
	setTwo := []uint32{3, 6, 9, 12, 15, 18, 21, 24, 27, 30}

	expected := []uint32{6, 12, 18, 24, 30}
	actual := FindIntersection(setOne, setTwo)

	for i, item := range expected {
		if item != actual[i] {
			t.Errorf("Error finding intersection in position %d. Expected %v. Actual %v", i, expected, actual)
		}
	}
}

func TestNewQuery(t *testing.T) {
	//TODO: Write tests for this function
}

func TestSearchForFrame(t *testing.T) {
	//TODO: Write tests for this function
}
