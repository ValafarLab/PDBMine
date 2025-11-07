package main

import (
	"strings"
	"testing"
)

func TestNewProteinRec(t *testing.T) {
	ProteinDatabase, err := NewProteinDB("Database.dat")

	proteinID := "1rwd"
	record, err := NewProteinRec(ProteinDatabase, proteinID)

	if err != nil {
		t.Errorf("Error finding %s. Expected no error.", proteinID)
	}

	if record.Name != strings.ToUpper(proteinID) {
		t.Errorf("Error in protein name. Expected %s but loaded %s", proteinID, record.Name)
	}

	// Test a non-existant ID returning an error
	proteinID = "1abc"
	_, err = NewProteinRec(ProteinDatabase, proteinID)

	if err == nil {
		t.Errorf("No error reported finding %s. Expected error.", proteinID)
	}
}
