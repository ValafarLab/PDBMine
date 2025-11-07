package main

import (
	"testing"
)

//TestLoadHeaderErrors tests the function that takes 16 bytes and creates a header record
//[65 110 103 108 101 68 97 116 2 0 4 76 32 0 6 0]
func TestLoadHeaderErrors(t *testing.T) {
	//Test invalid number of bytes - too small
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6}

	err := fakeDB.LoadHeader(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	fakeDB = new(ProteinDB)
	data = []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0, 5}

	err = fakeDB.LoadHeader(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB = new(ProteinDB)
	data = []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 77, 32, 0, 6, 0}

	err = fakeDB.LoadHeader(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadHeaderLittleEndian tests the function that takes 16 bytes and creates a header record
//[65 110 103 108 101 68 97 116 2 0 4 76 32 0 6 0]
func TestLoadHeaderLittleEndian(t *testing.T) {
	//Test valid number of bytes (little endian)
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}

	err := fakeDB.LoadHeader(data)

	if err != nil {
		t.Errorf("Error parsing header. No error expected")
	}

	if fakeDB.Header.Signature != "AngleDat" {
		t.Errorf("Unexpected signature in Header. Expected %s, received %s", "AngleDat", fakeDB.Header.Signature)
	}

	if fakeDB.Header.VersionInfo != "2.0.4" {
		t.Errorf("Unexpected version in Header. Expected %s, received %s", "2.0.4", fakeDB.Header.VersionInfo)
	}

	if fakeDB.Header.EndianType != "Little" {
		t.Errorf("Unexpected endianness in Header. Expected %s, received %s", "Little", fakeDB.Header.EndianType)
	}

	if fakeDB.Header.ManifestEntries != 6 {
		t.Errorf("Unexpected manifest entries in Header. Expected %d, received %d", 6, fakeDB.Header.ManifestEntries)
	}

	if fakeDB.Header.ManifestStart != 32 {
		t.Errorf("Unexpected manifest start in Header. Expected %d, received %d", 32, fakeDB.Header.ManifestStart)
	}
}

//TestLoadHeaderBigEndian tests the function that takes 16 bytes and creates a header record
//[65 110 103 108 101 68 97 116 2 0 4 66 0 32 0 6]
func TestLoadHeaderBigEndian(t *testing.T) {
	//Test valid number of bytes (big endian)
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}

	err := fakeDB.LoadHeader(data)

	if err != nil {
		t.Errorf("Error parsing header. No error expected")
	}

	if fakeDB.Header.Signature != "AngleDat" {
		t.Errorf("Unexpected signature in Header. Expected %s, received %s", "AngleDat", fakeDB.Header.Signature)
	}

	if fakeDB.Header.VersionInfo != "2.0.4" {
		t.Errorf("Unexpected version in Header. Expected %s, received %s", "2.0.4", fakeDB.Header.VersionInfo)
	}

	if fakeDB.Header.EndianType != "Big" {
		t.Errorf("Unexpected endianness in Header. Expected %s, received %s", "Big", fakeDB.Header.EndianType)
	}

	if fakeDB.Header.ManifestEntries != 6 {
		t.Errorf("Unexpected manifest entries in Header. Expected %d, received %d", 6, fakeDB.Header.ManifestEntries)
	}

	if fakeDB.Header.ManifestStart != 32 {
		t.Errorf("Unexpected manifest start in Header. Expected %d, received %d", 32, fakeDB.Header.ManifestStart)
	}
}

//TestLoadManifestEntryErrors tests the LoadManifestEntry function that takes 32 bytes and creates a manifest entry
//[74 117 109 112  84  97  98 108 101   0 0 0 0 0 0 0 0 0 16 0  87 239  0 0 224   0   0  0 0 0 0 0]
//[83 101 116  77 101 109  98 101 114 115 0 0 0 0 0 0 1 0  4 0   4 152  3 6  80 246  14  0 0 0 0 0]
//[80  68  66  68  97 116  97   0   0   0 0 0 0 0 0 0 2 0 12 0 161  69  2 0  96  86  29 24 0 0 0 0]
//[67 104  97 105 110 115   0   0   0   0 0 0 0 0 0 0 3 0 32 0 144  50  6 0 236 153  56 24 0 0 0 0]
//[68 105 104 101 100 114  97 108 115   0 0 0 0 0 0 0 4 0  4 0 223 130 53 7 236 235 254 24 0 0 0 0]
//[82 101 115 105 100 117 101 115   0   0 0 0 0 0 0 0 5 0  1 0   4 152  3 6 104 247 212 53 0 0 0 0]
func TestLoadManifestEntryErrors(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test invalid number of bytes - too small
	data = []byte{80, 68, 66, 68, 97, 116, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 12, 0, 161, 69, 2, 0, 96, 86, 29, 24, 0, 0, 0}
	err := fakeDB.LoadManifestEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	data = []byte{80, 68, 66, 68, 97, 116, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 12, 0, 161, 69, 2, 0, 96, 86, 29, 24, 0, 0, 0, 0, 0}
	err = fakeDB.LoadManifestEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB.Header.EndianType = "Junk"
	data = []byte{80, 68, 66, 68, 97, 116, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 12, 0, 161, 69, 2, 0, 96, 86, 29, 24, 0, 0, 0, 0}
	err = fakeDB.LoadManifestEntry(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadManifestEntryLittleEndian tests the LoadManifestEntry function that takes 32 bytes and creates a manifest entry
//[80  68  66  68  97 116  97   0   0   0 0 0 0 0 0 0 2 0 12 0 161  69  2 0  96  86  29 24 0 0 0 0]
func TestLoadManifestEntryLittleEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test valid number of bytes (little endian)
	data = []byte{80, 68, 66, 68, 97, 116, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 12, 0, 161, 69, 2, 0, 96, 86, 29, 24, 0, 0, 0, 0}

	err := fakeDB.LoadManifestEntry(data)

	if err != nil {
		t.Errorf("Error parsing manifest entry. No error expected %v", err)
	}

	if len(fakeDB.Manifest) != 1 {
		t.Errorf("Expected only %d manifest entries but have %d", 1, len(fakeDB.Manifest))
	}

	if fakeDB.Manifest[0].Label != "PDBData" {
		t.Errorf("Unexpected label in Manifest Entry. Expected %s, received %s", "PDBData", fakeDB.Manifest[0].Label)
	}

	if fakeDB.Manifest[0].SectionID != 2 {
		t.Errorf("Unexpected section ID in Manifest Entry. Expected %d, received %d", 2, fakeDB.Manifest[0].SectionID)
	}

	if fakeDB.Manifest[0].Size != 12 {
		t.Errorf("Unexpected size in Manifest Entry. Expected %d, received %d", 12, fakeDB.Manifest[0].Size)
	}

	if fakeDB.Manifest[0].NumEntries != 148897 {
		t.Errorf("Unexpected num entries in Manifest Entry. Expected %d, received %d", 148897, fakeDB.Manifest[0].NumEntries)
	}

	if fakeDB.Manifest[0].Start != 404575840 {
		t.Errorf("Unexpected start in Manifest Entry. Expected %d, received %d", 404575840, fakeDB.Manifest[0].Start)
	}
}

//TestLoadManifestEntryBigEndian tests the LoadManifestEntry function that takes 32 bytes and creates a manifest entry
//[80  68  66  68  97 116  97   0   0   0 0 0 0 0 0 0 2 0 12 0 161  69  2 0  96  86  29 24 0 0 0 0]
func TestLoadManifestEntryBigEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}
	fakeDB.LoadHeader(data)

	//Test valid number of bytes (little endian)
	data = []byte{80, 68, 66, 68, 97, 116, 97, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 12, 0, 2, 69, 161, 0, 0, 0, 0, 24, 29, 86, 96}

	err := fakeDB.LoadManifestEntry(data)

	if err != nil {
		t.Errorf("Error parsing manifest entry. No error expected %v", err)
	}

	if len(fakeDB.Manifest) != 1 {
		t.Errorf("Expected only %d manifest entries but have %d", 1, len(fakeDB.Manifest))
	}

	if fakeDB.Manifest[0].Label != "PDBData" {
		t.Errorf("Unexpected label in Manifest Entry. Expected %s, received %s", "PDBData", fakeDB.Manifest[0].Label)
	}

	if fakeDB.Manifest[0].SectionID != 2 {
		t.Errorf("Unexpected section ID in Manifest Entry. Expected %d, received %d", 2, fakeDB.Manifest[0].SectionID)
	}

	if fakeDB.Manifest[0].Size != 12 {
		t.Errorf("Unexpected size in Manifest Entry. Expected %d, received %d", 12, fakeDB.Manifest[0].Size)
	}

	if fakeDB.Manifest[0].NumEntries != 148897 {
		t.Errorf("Unexpected num entries in Manifest Entry. Expected %d, received %d", 148897, fakeDB.Manifest[0].NumEntries)
	}

	if fakeDB.Manifest[0].Start != 404575840 {
		t.Errorf("Unexpected start in Manifest Entry. Expected %d, received %d", 404575840, fakeDB.Manifest[0].Start)
	}
}

//TestLoadJumpTableErrors tests the function that loads the Jump Table Entries into the map
//[69 0 196 13 1 0 0 0 255 148 3 6 0 0 0 0]
//[73 0 196 13 2 0 0 0   0 149 3 6 0 0 0 0]
func TestLoadJumpTableEntryErrors(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test invalid number of bytes - too small
	data = []byte{73, 0, 196, 13, 2, 0, 0, 0, 0, 149, 3, 6, 0, 0, 0}
	err := fakeDB.LoadJumpTableEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	data = []byte{73, 0, 196, 13, 2, 0, 0, 0, 0, 149, 3, 6, 0, 0, 0, 0, 0}
	err = fakeDB.LoadJumpTableEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB.Header.EndianType = "Junk"

	data = []byte{73, 0, 196, 13, 2, 0, 0, 0, 0, 149, 3, 6, 0, 0, 0, 0}
	err = fakeDB.LoadJumpTableEntry(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadJumpTableLittleEndian tests the function that loads the Jump Table Entries into the map
//[73 0 196 13 2 0 0 0   0 149 3 6 0 0 0 0]
func TestLoadJumpTableEntryLittleEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	fakeDB.JumpTable = make(map[JumpKey]JumpTableEntry, 1)

	//Test invalid number of bytes - too small
	data = []byte{73, 0, 196, 13, 2, 0, 0, 0, 0, 149, 3, 6, 0, 0, 0, 0}
	err := fakeDB.LoadJumpTableEntry(data)

	if err != nil {
		t.Errorf("Error parsing jump table entry. No error expected %v.", err)
	}

	jk := new(JumpKey)
	jk.ResidueName = "I"
	jk.ResIndex = uint32(3524)

	jte := new(JumpTableEntry)
	jte.ResidueName = "I"
	jte.ResIndex = uint16(3524)
	jte.SetNumMembers = 2
	jte.SetStartIndex = 100898048

	entry, found := fakeDB.JumpTable[*jk]
	if !found {
		t.Errorf("Jump Entry record was not found.")
	}

	if entry.ResIndex != jte.ResIndex {
		t.Errorf("ResIndex mismatch. Expected %d, received %d", jte.ResIndex, entry.ResIndex)
	}

	if entry.ResidueName != jte.ResidueName {
		t.Errorf("ResidueName mismatch. Expected %s, received %s", jte.ResidueName, entry.ResidueName)
	}

	if entry.SetNumMembers != jte.SetNumMembers {
		t.Errorf("SetNumMembers mismatch. Expected %d, received %d", jte.SetNumMembers, entry.SetNumMembers)
	}

	if entry.SetStartIndex != jte.SetStartIndex {
		t.Errorf("SetStartIndex mismatch. Expected %d, received %d", jte.SetStartIndex, entry.SetStartIndex)
	}
}

//TestLoadJumpTableBigEndian tests the function that loads the Jump Table Entries into the map
//[73 0 13 196 0 0 0 2 0 0 0 0 6 3 149 0  ]
func TestLoadJumpTableEntryBigEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}
	fakeDB.LoadHeader(data)

	fakeDB.JumpTable = make(map[JumpKey]JumpTableEntry, 1)

	//Test invalid number of bytes - too small
	data = []byte{73, 0, 13, 196, 0, 0, 0, 2, 0, 0, 0, 0, 6, 3, 149, 0}
	err := fakeDB.LoadJumpTableEntry(data)

	if err != nil {
		t.Errorf("Error parsing jump table entry. No error expected %v.", err)
	}

	jk := new(JumpKey)
	jk.ResidueName = "I"
	jk.ResIndex = uint32(3524)

	jte := new(JumpTableEntry)
	jte.ResidueName = "I"
	jte.ResIndex = uint16(3524)
	jte.SetNumMembers = 2
	jte.SetStartIndex = 100898048

	entry, found := fakeDB.JumpTable[*jk]
	if !found {
		t.Errorf("Jump Entry record was not found.")
	}

	if entry.ResIndex != jte.ResIndex {
		t.Errorf("ResIndex mismatch. Expected %d, received %d", jte.ResIndex, entry.ResIndex)
	}

	if entry.ResidueName != jte.ResidueName {
		t.Errorf("ResidueName mismatch. Expected %s, received %s", jte.ResidueName, entry.ResidueName)
	}

	if entry.SetNumMembers != jte.SetNumMembers {
		t.Errorf("SetNumMembers mismatch. Expected %d, received %d", jte.SetNumMembers, entry.SetNumMembers)
	}

	if entry.SetStartIndex != jte.SetStartIndex {
		t.Errorf("SetStartIndex mismatch. Expected %d, received %d", jte.SetStartIndex, entry.SetStartIndex)
	}
}

//TestLoadPDBEntryErrors tests the function that loads the PDB Entries into the map
//[49 82 87 68 1 0 0 0 47 173 0 0]
func TestLoadPDBEntryErrors(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test invalid number of bytes - too small
	data = []byte{49, 82, 87, 68, 1, 0, 0, 0, 47, 173, 0}
	err := fakeDB.LoadPDBEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	data = []byte{49, 82, 87, 68, 1, 0, 0, 0, 47, 173, 0, 0, 0}
	err = fakeDB.LoadPDBEntry(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB.Header.EndianType = "Junk"

	data = []byte{49, 82, 87, 68, 1, 0, 0, 0, 47, 173, 0, 0}
	err = fakeDB.LoadPDBEntry(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadPDBEntryLittleEndian tests the function that loads the PDB Entries into the map
//[49 82 87 68 1 0 0 0 47 173 0 0]
func TestLoadPDBEntryLittleEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	fakeDB.PDBData = make(map[string]PDBRec, 1)

	data = []byte{49, 82, 87, 68, 1, 0, 0, 0, 47, 173, 0, 0}
	err := fakeDB.LoadPDBEntry(data)

	if err != nil {
		t.Errorf("Error parsing pdb entry. No error expected %v.", err)
	}

	pdb := new(PDBRec)
	pdb.Name = "1RWD"
	pdb.NumChains = 1
	pdb.ChainIndex = 44335

	entry, found := fakeDB.PDBData[pdb.Name]

	if !found {
		t.Errorf("PDB Entry record was not found.")
	}

	if entry.Name != pdb.Name {
		t.Errorf("Name mismatch. Expected %s, received %s", pdb.Name, entry.Name)
	}

	if entry.NumChains != pdb.NumChains {
		t.Errorf("NumChains mismatch. Expected %d, received %d", pdb.NumChains, entry.NumChains)
	}

	if entry.ChainIndex != pdb.ChainIndex {
		t.Errorf("ChainIndex mismatch. Expected %d, received %d", pdb.ChainIndex, entry.ChainIndex)
	}
}

//TestLoadPDBEntryBigEndian tests the function that loads the PDB Entries into the map
//[49 82 87 68 0 0 0 1 0 0 173 47]
func TestLoadPDBEntryBigEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}
	fakeDB.LoadHeader(data)

	fakeDB.PDBData = make(map[string]PDBRec, 1)

	data = []byte{49, 82, 87, 68, 0, 0, 0, 1, 0, 0, 173, 47}
	err := fakeDB.LoadPDBEntry(data)

	if err != nil {
		t.Errorf("Error parsing pdb entry. No error expected %v.", err)
	}

	pdb := new(PDBRec)
	pdb.Name = "1RWD"
	pdb.NumChains = 1
	pdb.ChainIndex = 44335

	entry, found := fakeDB.PDBData[pdb.Name]

	if !found {
		t.Errorf("PDB Entry record was not found.")
	}

	if entry.Name != pdb.Name {
		t.Errorf("Name mismatch. Expected %s, received %s", pdb.Name, entry.Name)
	}

	if entry.NumChains != pdb.NumChains {
		t.Errorf("NumChains mismatch. Expected %d, received %d", pdb.NumChains, entry.NumChains)
	}

	if entry.ChainIndex != pdb.ChainIndex {
		t.Errorf("ChainIndex mismatch. Expected %d, received %d", pdb.ChainIndex, entry.ChainIndex)
	}
}

//TestLoadChainErrors tests the function that adds a chain to the chains array
//[49 82 87 68 224 77 0 0 65 0 1 0 50 0 50 0 193 39 208 0 0 0 0 0 174 205 155 0 0 0 0 0]
func TestLoadChainErrors(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test invalid number of bytes - too small
	data = []byte{49, 82, 87, 68, 224, 77, 0, 0, 65, 0, 1, 0, 50, 0, 50, 0, 193, 39, 208, 0, 0, 0, 0, 0, 174, 205, 155, 0, 0, 0, 0}
	err := fakeDB.LoadChain(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	data = []byte{49, 82, 87, 68, 224, 77, 0, 0, 65, 0, 1, 0, 50, 0, 50, 0, 193, 39, 208, 0, 0, 0, 0, 0, 174, 205, 155, 0, 0, 0, 0, 0, 0}
	err = fakeDB.LoadChain(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB.Header.EndianType = "Junk"

	data = []byte{49, 82, 87, 68, 224, 77, 0, 0, 65, 0, 1, 0, 50, 0, 50, 0, 193, 39, 208, 0, 0, 0, 0, 0, 174, 205, 155, 0, 0, 0, 0, 0}
	err = fakeDB.LoadChain(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadChainLittleEndian tests the function that adds a chain to the chains array
//[49 82 87 68 224 77 0 0 65 0 1 0 50 0 50 0 193 39 208 0 0 0 0 0 174 205 155 0 0 0 0 0]
func TestLoadChainLittleEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	fakeDB.Chains = make([]Chain, 0)

	data = []byte{49, 82, 87, 68, 224, 77, 0, 0, 65, 0, 1, 0, 50, 0, 50, 0, 193, 39, 208, 0, 0, 0, 0, 0, 174, 205, 155, 0, 0, 0, 0, 0}
	err := fakeDB.LoadChain(data)

	if err != nil {
		t.Errorf("Error parsing chain entry. No error expected. Error: %v.", err)
	}

	chain := new(Chain)
	chain.SourcePDB = "1RWD"
	chain.SourceIndex = 19936
	chain.ChainID = "A"
	chain.NumModels = 1
	chain.NumResidues = 50
	chain.ModelLength = 50
	chain.ModelIndex = 13641665
	chain.ResidueIndex = 10210734

	if len(fakeDB.Chains) != 1 {
		t.Errorf("Expected only %d chain entries but have %d", 1, len(fakeDB.Chains))
	}

	if fakeDB.Chains[0].SourcePDB != chain.SourcePDB {
		t.Errorf("Unexpected Source PDB in Chain Entry. Expected %s, received %s", chain.SourcePDB, fakeDB.Chains[0].SourcePDB)
	}

	if fakeDB.Chains[0].SourceIndex != chain.SourceIndex {
		t.Errorf("Unexpected Source Index in Chain Entry. Expected %d, received %d", chain.SourceIndex, fakeDB.Chains[0].SourceIndex)
	}

	if fakeDB.Chains[0].ChainID != chain.ChainID {
		t.Errorf("Unexpected Chain ID in Chain Entry. Expected %s, received %s", chain.ChainID, fakeDB.Chains[0].ChainID)
	}

	if fakeDB.Chains[0].NumModels != chain.NumModels {
		t.Errorf("Unexpected Num Models in Chain Entry. Expected %d, received %d", chain.NumModels, fakeDB.Chains[0].NumModels)
	}

	if fakeDB.Chains[0].NumResidues != chain.NumResidues {
		t.Errorf("Unexpected Num Residues in Chain Entry. Expected %d, received %d", chain.NumResidues, fakeDB.Chains[0].NumResidues)
	}

	if fakeDB.Chains[0].ModelLength != chain.ModelLength {
		t.Errorf("Unexpected Model Length in Chain Entry. Expected %d, received %d", chain.ModelLength, fakeDB.Chains[0].ModelLength)
	}

	if fakeDB.Chains[0].ModelIndex != chain.ModelIndex {
		t.Errorf("Unexpected Model Index in Chain Entry. Expected %d, received %d", chain.ModelIndex, fakeDB.Chains[0].ModelIndex)
	}

	if fakeDB.Chains[0].ResidueIndex != chain.ResidueIndex {
		t.Errorf("Unexpected Residue Index in Chain Entry. Expected %d, received %d", chain.ResidueIndex, fakeDB.Chains[0].ResidueIndex)
	}
}

//TestLoadChainBigEndian tests the function that adds a chain to the chains array
//[49 82 87 68 224 77 0 0 65 0 1 0 50 0 50 0 193 39 208 0 0 0 0 0 174 205 155 0 0 0 0 0]
func TestLoadChainBigEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}
	fakeDB.LoadHeader(data)

	fakeDB.Chains = make([]Chain, 0)

	data = []byte{49, 82, 87, 68, 0, 0, 77, 224, 65, 0, 0, 1, 0, 50, 0, 50, 0, 0, 0, 0, 0, 208, 39, 193, 0, 0, 0, 0, 0, 155, 205, 174}
	err := fakeDB.LoadChain(data)

	if err != nil {
		t.Errorf("Error parsing chain entry. No error expected. Error: %v.", err)
	}

	chain := new(Chain)
	chain.SourcePDB = "1RWD"
	chain.SourceIndex = 19936
	chain.ChainID = "A"
	chain.NumModels = 1
	chain.NumResidues = 50
	chain.ModelLength = 50
	chain.ModelIndex = 13641665
	chain.ResidueIndex = 10210734

	if len(fakeDB.Chains) != 1 {
		t.Errorf("Expected only %d chain entries but have %d", 1, len(fakeDB.Chains))
	}

	if fakeDB.Chains[0].SourcePDB != chain.SourcePDB {
		t.Errorf("Unexpected Source PDB in Chain Entry. Expected %s, received %s", chain.SourcePDB, fakeDB.Chains[0].SourcePDB)
	}

	if fakeDB.Chains[0].SourceIndex != chain.SourceIndex {
		t.Errorf("Unexpected Source Index in Chain Entry. Expected %d, received %d", chain.SourceIndex, fakeDB.Chains[0].SourceIndex)
	}

	if fakeDB.Chains[0].ChainID != chain.ChainID {
		t.Errorf("Unexpected Chain ID in Chain Entry. Expected %s, received %s", chain.ChainID, fakeDB.Chains[0].ChainID)
	}

	if fakeDB.Chains[0].NumModels != chain.NumModels {
		t.Errorf("Unexpected Num Models in Chain Entry. Expected %d, received %d", chain.NumModels, fakeDB.Chains[0].NumModels)
	}

	if fakeDB.Chains[0].NumResidues != chain.NumResidues {
		t.Errorf("Unexpected Num Residues in Chain Entry. Expected %d, received %d", chain.NumResidues, fakeDB.Chains[0].NumResidues)
	}

	if fakeDB.Chains[0].ModelLength != chain.ModelLength {
		t.Errorf("Unexpected Model Length in Chain Entry. Expected %d, received %d", chain.ModelLength, fakeDB.Chains[0].ModelLength)
	}

	if fakeDB.Chains[0].ModelIndex != chain.ModelIndex {
		t.Errorf("Unexpected Model Index in Chain Entry. Expected %d, received %d", chain.ModelIndex, fakeDB.Chains[0].ModelIndex)
	}

	if fakeDB.Chains[0].ResidueIndex != chain.ResidueIndex {
		t.Errorf("Unexpected Residue Index in Chain Entry. Expected %d, received %d", chain.ResidueIndex, fakeDB.Chains[0].ResidueIndex)
	}
}

//TestLoadMemberErrors tests the errors that can occur in the LoadMember function that adds an integer to the SetMembers array
//[48 6 5 0]
func TestLoadMemberErrors(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	//Test invalid number of bytes - too small
	data = []byte{48, 6, 5}
	err := fakeDB.LoadMember(data)

	if err == nil {
		t.Errorf("No error for byte array too small. Error expected")
	}

	//Test invalid number of bytes - too large
	data = []byte{48, 6, 5, 0, 7}
	err = fakeDB.LoadMember(data)

	if err == nil {
		t.Errorf("No error for byte array too large. Error expected")
	}

	//Test unknown Endianness
	fakeDB.Header.EndianType = "Junk"

	data = []byte{48, 6, 5, 0}
	err = fakeDB.LoadMember(data)

	if err == nil {
		t.Errorf("No error for invalid endianness. Error expected")
	}
}

//TestLoadMemberLittleEndian tests the LoadMember function that adds an integer to the SetMembers array
//[48 6 5 0]
func TestLoadMemberLittleEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 76, 32, 0, 6, 0}
	fakeDB.LoadHeader(data)

	fakeDB.SetMembers = make([]uint32, 0)

	data = []byte{48, 6, 5, 0}
	err := fakeDB.LoadMember(data)

	if err != nil {
		t.Errorf("Error parsing chain entry. No error expected. Error: %v.", err)
	}

	if len(fakeDB.SetMembers) != 1 {
		t.Errorf("Expected only %d set member entries but have %d", 1, len(fakeDB.Chains))
	}

	expected := uint32(329264)
	if fakeDB.SetMembers[0] != expected {
		t.Errorf("Unexpected Set Member. Expected %d, received %d", expected, fakeDB.SetMembers[0])
	}
}

//TestLoadMemberBigEndian tests the LoadMember function that adds an integer to the SetMembers array
//[0 5 6 48]
func TestLoadMemberBigEndian(t *testing.T) {
	//Setup the Header
	fakeDB := new(ProteinDB)
	data := []byte{65, 110, 103, 108, 101, 68, 97, 116, 2, 0, 4, 66, 0, 32, 0, 6}
	fakeDB.LoadHeader(data)

	fakeDB.SetMembers = make([]uint32, 0)

	data = []byte{0, 5, 6, 48}
	err := fakeDB.LoadMember(data)

	if err != nil {
		t.Errorf("Error parsing chain entry. No error expected. Error: %v.", err)
	}

	if len(fakeDB.SetMembers) != 1 {
		t.Errorf("Expected only %d set member entries but have %d", 1, len(fakeDB.Chains))
	}

	expected := uint32(329264)
	if fakeDB.SetMembers[0] != expected {
		t.Errorf("Unexpected Set Member. Expected %d, received %d", expected, fakeDB.SetMembers[0])
	}
}

//TestLoadDihedralErrors tests the errors that can occur in the LoadDihedral function that adds an Phi and Psi angles to the Dihedrals array
//[ 32 28  61 20]
//[103  8 111 19]
//[ 90 10 234 19]
//[213 10 147 20]
//[159 11 165 12]
//[188 11 108 12]
//[ 97 11 126 12]
//[149 11 160 12]
//[136 11 143 12]
//[136 11 107 12]
func TestLoadDihedralError(t *testing.T) {

}

//TestLoadDihedralLittleEndian tests the LoadDihedral function that adds an Phi and Psi angles to the Dihedrals array
//[ 32 28  61 20]
//[103  8 111 19]
//[ 90 10 234 19]
//[213 10 147 20]
//[159 11 165 12]
//[188 11 108 12]
//[ 97 11 126 12]
//[149 11 160 12]
//[136 11 143 12]
//[136 11 107 12]
func TestLoadDihedralLittleEndian(t *testing.T) {

}

//TestLoadDihedralBigEndian tests the LoadDihedral function that adds an Phi and Psi angles to the Dihedrals array
//[ 32 28  61 20]
//[103  8 111 19]
//[ 90 10 234 19]
//[213 10 147 20]
//[159 11 165 12]
//[188 11 108 12]
//[ 97 11 126 12]
//[149 11 160 12]
//[136 11 143 12]
//[136 11 107 12]
func TestLoadDihedralBigEndian(t *testing.T) {

}

//TestNewProteinDB tests the function that loads the entire database into memory. This will just be basic checks since the individual functions were already tested above.
func TestNewProteinDB(t *testing.T) {
	pdb, err := NewProteinDB("Database.dat")

	if err != nil {
		t.Errorf("Error loading database. Expected no error.")
	}

	expectedDBName := "AngleDat"
	expectedDBVersion := "2.0.4"
	//init() is run by default, so check and see if the database is up and available
	if pdb.Header.Signature != expectedDBName {
		t.Errorf("Error loading Database. Expected header name of %s but header name is %s", expectedDBName, pdb.Header.Signature)
	}

	if pdb.Header.VersionInfo != expectedDBVersion {
		t.Errorf("Error loading Database. Expected version of %s but version is %s", expectedDBName, pdb.Header.Signature)
	}

	if int(pdb.Header.ManifestEntries) != len(pdb.Manifest) {
		t.Errorf("Manifest Entry mismatch. Expected %d records from header but actually have %d records in manifest array", pdb.Header.ManifestEntries, len(pdb.Manifest))
	}

}
