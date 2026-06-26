package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	//Chi Router
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	//Redis connection
	"github.com/gomodule/redigo/redis"
)

const (
	//HeaderSize is the size of the Header record
	HeaderSize int = 16
	//ManifestEntrySize is the size of a Manifest entry
	ManifestEntrySize int = 32
	//JumpTableEntrySize is the size of a Jump Table entry
	JumpTableEntrySize int = 16
	//PDBEntrySize is the size of a PDB Table entry
	PDBEntrySize int = 12
	//ChainEntrySize is the size of a Chain entry
	ChainEntrySize int = 32
	//SetMemberSize is the size of a Set Member
	SetMemberSize int = 4
	//DihedralSize is the size of a Dihedral which includes both Phi and PSi
	DihedralSize int = 4
)

//HeaderRec is the header record for the database
type HeaderRec struct {
	Signature       string `json:"signature"`
	VersionInfo     string `json:"versionInfo"`
	EndianType      string `json:"endianType"`
	ManifestStart   uint16 `json:"manifestStart"`
	ManifestEntries uint16 `json:"manifestEntries"`
}

//LoadHeader parses an array of UTF8 bytes into the Header record
// The Layout of a Header Record is:
// Size (bytes) | Description
// --------------------------
//          08  | Signature: The signature or identifier of the file
//          03  | VersionInfo: The version of the file
//          01  | EndianType: Endianness of numbers in the file. Should be (L)ittle or (B)ig
//          02  | ManifestStart: The starting position of the Manifest
//          02  | ManifestEntries: The number of Manifest Entries in the file
func (db *ProteinDB) LoadHeader(data []byte) error {
	if len(data) == HeaderSize {
		//Copy the signature, but strip out null bytes
		db.Header.Signature = strings.Trim(string(data[0:8]), "\000")

		//Set the version as Major/Minor/Build
		db.Header.VersionInfo = fmt.Sprintf("%d.%d.%d", data[8], data[9], data[10])

		//Set Endianness, Manifest Start and MAnifest Entries based on the Endianness
		switch data[11] {
		case "L"[0]:
			db.Header.EndianType = "Little"
			//Copy the Manifest Start and Manifest Entries into a 2 byte array. Go doesn't like a slice for the conversion from Binary
			temp := make([]byte, 2)
			copy(temp, data[12:14])
			db.Header.ManifestStart = binary.LittleEndian.Uint16(temp)
			copy(temp, data[14:16])
			db.Header.ManifestEntries = binary.LittleEndian.Uint16(temp)
		case "B"[0]:
			db.Header.EndianType = "Big"
			//Copy the Manifest Start and Manifest Entries
			temp := make([]byte, 2)
			copy(temp, data[12:14])
			db.Header.ManifestStart = binary.BigEndian.Uint16(temp)
			copy(temp, data[14:16])
			db.Header.ManifestEntries = binary.BigEndian.Uint16(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected. Expected 'B' or 'L' but received %x", data[11])
		}
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), HeaderSize)
	}

	return nil
}

//ManifestEntry is the structure of one entry in the file manifest
type ManifestEntry struct {
	Label      string `json:"label"`
	SectionID  uint16 `json:"sectionID"`
	Size       uint16 `json:"size"`
	NumEntries uint32 `json:"numEntries"`
	Start      uint64 `json:"start"`
}

//LoadManifestEntry parses a UTF8 byte array into a Manifest Entry for the Manifest
// The Layout of a Manifest Entry is:
// Size (bytes) | Description
// --------------------------
//          16  | Label: The identifier for this manifest entry
//          02  | Section ID: An ID for this section
//          02  | Size: The size of each individual record in the section pointed to by this manifest entry
//          04  | NumEntries: The number of records in the section pointed to by this manifest entry
//          08  | Start: The offset in the file where this section starts
func (db *ProteinDB) LoadManifestEntry(data []byte) error {
	if len(data) == ManifestEntrySize {
		me := new(ManifestEntry)

		//Copy the label, but strip out null bytes
		me.Label = strings.Trim(string(data[0:16]), "\000")

		switch db.Header.EndianType {
		case "Little":
			//Get the Section ID
			temp := make([]byte, 2)
			copy(temp, data[16:18])
			me.SectionID = binary.LittleEndian.Uint16(temp)
			//Get the Size
			copy(temp, data[18:20])
			me.Size = binary.LittleEndian.Uint16(temp)
			//Get the number of entries
			temp = make([]byte, 4)
			copy(temp, data[20:24])
			me.NumEntries = binary.LittleEndian.Uint32(temp)
			//Get the starting offset
			temp = make([]byte, 8)
			copy(temp, data[24:32])
			me.Start = binary.LittleEndian.Uint64(temp)
		case "Big":
			//Get the Section ID
			temp := make([]byte, 2)
			copy(temp, data[16:18])
			me.SectionID = binary.BigEndian.Uint16(temp)
			//Get the Size
			copy(temp, data[18:20])
			me.Size = binary.BigEndian.Uint16(temp)
			//Get the number of entries
			temp = make([]byte, 4)
			copy(temp, data[20:24])
			me.NumEntries = binary.BigEndian.Uint32(temp)
			//Get the starting offset
			temp = make([]byte, 8)
			copy(temp, data[24:32])
			me.Start = binary.BigEndian.Uint64(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}

		db.Manifest = append(db.Manifest, *me)

	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), ManifestEntrySize)
	}

	return nil
}

//JumpKey is a key into the Jump Table
type JumpKey struct {
	ResIndex    uint32 `json:"resIndex"`
	ResidueName string `json:"residueName"`
}

//JumpTableEntry is the structure of an entry in the jump table
type JumpTableEntry struct {
	ResidueName   string `json:"residueName"`
	Padding       byte   `json:"padding"`
	ResIndex      uint16 `json:"resIndex"` //TODO: Check - should this be the same size as the ResIndex in the JumpKey
	SetNumMembers uint32 `json:"setNumMembers"`
	SetStartIndex uint64 `json:"setStartIndex"`
}

//LoadJumpTableEntry parses a UTF8 byte array into one entry for the Jump Table
// The Layout of a Jump Table Entry is:
// Size (bytes) | Description
// --------------------------
//          01  | Residue Name: The residue name for this entry
//          01  | Padding: Padding to make everything line up nicely
//          02  | ResIndex: The index of the residue in the residues section
//          04  | SetNumMembers: The number of members in the set
//          08  | SetStartIndex: The offset in the file where the set starts
func (db *ProteinDB) LoadJumpTableEntry(data []byte) error {
	//even though the size is pulled dynamically from the manifest entry, this function has to assume a particular size to parse correctly
	if len(data) == JumpTableEntrySize {
		jte := new(JumpTableEntry)
		jk := new(JumpKey)

		jte.ResidueName = string(data[0])
		jk.ResidueName = string(data[0])

		jte.Padding = data[1]

		switch db.Header.EndianType {
		case "Little":
			temp := make([]byte, 2)
			copy(temp, data[2:4])
			jte.ResIndex = binary.LittleEndian.Uint16(temp)
			jk.ResIndex = uint32(jte.ResIndex)
			temp = make([]byte, 4)
			copy(temp, data[4:8])
			jte.SetNumMembers = binary.LittleEndian.Uint32(temp)
			temp = make([]byte, 8)
			copy(temp, data[8:16])
			jte.SetStartIndex = binary.LittleEndian.Uint64(temp)
		case "Big":
			temp := make([]byte, 2)
			copy(temp, data[2:4])
			jte.ResIndex = binary.BigEndian.Uint16(temp)
			jk.ResIndex = uint32(jte.ResIndex)
			temp = make([]byte, 4)
			copy(temp, data[4:8])
			jte.SetNumMembers = binary.BigEndian.Uint32(temp)
			temp = make([]byte, 8)
			copy(temp, data[8:16])
			jte.SetStartIndex = binary.BigEndian.Uint64(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}

		db.JumpTable[*jk] = *jte
		if jk.ResIndex > db.JumpIndexMax {
			db.JumpIndexMax = jk.ResIndex
		}
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), JumpTableEntrySize)
	}

	return nil
}

//PDBRec is the structure of a PDB Record from the PDB database
type PDBRec struct {
	Name       string `json:"name"`
	NumChains  uint32 `json:"numChains"`
	ChainIndex uint32 `json:"chainIndex"`
}

//LoadPDBEntry parses a UTF8 byte array into one entry for the PDB map
// The Layout of a PDB Entry is:
// Size (bytes) | Description
// --------------------------
//          04  | Name: The name of the protein (i.e. 1RWD)
//          04  | NumChains: The number of chains in the protein. These will be read from the Chains array.
//          04  | ChainIndex: The index to start reading chains from for the protein
func (db *ProteinDB) LoadPDBEntry(data []byte) error {
	if len(data) == PDBEntrySize {
		pdb := new(PDBRec)

		pdb.Name = string(data[0:4])

		switch db.Header.EndianType {
		case "Little":
			temp := make([]byte, 4)
			copy(temp, data[4:8])
			pdb.NumChains = binary.LittleEndian.Uint32(temp)
			copy(temp, data[8:12])
			pdb.ChainIndex = binary.LittleEndian.Uint32(temp)
		case "Big":
			temp := make([]byte, 4)
			copy(temp, data[4:8])
			pdb.NumChains = binary.BigEndian.Uint32(temp)
			copy(temp, data[8:12])
			pdb.ChainIndex = binary.BigEndian.Uint32(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}

		db.PDBData[pdb.Name] = *pdb
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), PDBEntrySize)
	}

	return nil
}

//Chain is the structure of a Residue Chain associated with a protein from the PDB database
type Chain struct {
	SourcePDB    string `json:"sourcePDB"`
	SourceIndex  uint32 `json:"sourceIndex"`
	ChainID      string `json:"chainID"`
	Padding      byte   `json:"padding"`
	NumModels    uint16 `json:"numModels"`
	NumResidues  uint16 `json:"numResidues"`
	ModelLength  uint16 `json:"modelLength"`
	ModelIndex   uint64 `json:"modelIndex"`
	ResidueIndex uint64 `json:"residueIndex"`
}

//LoadChain parses a UTF8 byte array into one chain
// The Layout of a Chain is:
// Size (bytes) | Description
// --------------------------
//          04  | SourcePDB: The name of the PDB that this chain belongs to.
//          04  | SourceIndex: The index of the PDB array that this chain belongs to. Since the PDB's are being stored in a map, this is less useful.
//          01  | ChainID: The ID of the chain. This is a one charcter letter.
//			01  | Padding: Some padding to make sure everything lines up.
//			02  | NumModels: The number of models.
//			02  | NumResidues: The number of residues in the chain.
//			02  | ModelLength: The length of the model.
//			08  | ModelIndex: The index of the model.
//			08  | ResidueIndex: The index of the residue.
func (db *ProteinDB) LoadChain(data []byte) error {
	if len(data) == ChainEntrySize {
		chain := new(Chain)

		chain.SourcePDB = string(data[0:4])
		chain.ChainID = string(data[8])
		chain.Padding = data[9]

		switch db.Header.EndianType {
		case "Little":
			temp := make([]byte, 4)
			copy(temp, data[4:8])
			chain.SourceIndex = binary.LittleEndian.Uint32(temp)
			temp = make([]byte, 2)
			copy(temp, data[10:12])
			chain.NumModels = binary.LittleEndian.Uint16(temp)
			copy(temp, data[12:14])
			chain.NumResidues = binary.LittleEndian.Uint16(temp)
			copy(temp, data[14:16])
			chain.ModelLength = binary.LittleEndian.Uint16(temp)
			temp = make([]byte, 8)
			copy(temp, data[16:24])
			chain.ModelIndex = binary.LittleEndian.Uint64(temp)
			copy(temp, data[24:32])
			chain.ResidueIndex = binary.LittleEndian.Uint64(temp)
		case "Big":
			temp := make([]byte, 4)
			copy(temp, data[4:8])
			chain.SourceIndex = binary.BigEndian.Uint32(temp)
			temp = make([]byte, 2)
			copy(temp, data[10:12])
			chain.NumModels = binary.BigEndian.Uint16(temp)
			copy(temp, data[12:14])
			chain.NumResidues = binary.BigEndian.Uint16(temp)
			copy(temp, data[14:16])
			chain.ModelLength = binary.BigEndian.Uint16(temp)
			temp = make([]byte, 8)
			copy(temp, data[16:24])
			chain.ModelIndex = binary.BigEndian.Uint64(temp)
			copy(temp, data[24:32])
			chain.ResidueIndex = binary.BigEndian.Uint64(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}

		db.Chains = append(db.Chains, *chain)
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), ChainEntrySize)
	}

	return nil
}

//LoadMember parses a UTF8 byte array into one chain
// Size (bytes) | Description
// --------------------------
//          04  | SetMember: a 4 byte integer
func (db *ProteinDB) LoadMember(data []byte) error {
	if len(data) == SetMemberSize {
		mem := uint32(0)
		temp := make([]byte, 4)
		copy(temp, data[0:4])
		switch db.Header.EndianType {
		case "Little":
			mem = binary.LittleEndian.Uint32(temp)
		case "Big":
			mem = binary.BigEndian.Uint32(temp)
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}
		db.SetMembers = append(db.SetMembers, mem)
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), SetMemberSize)
	}

	return nil
}

//Dihedral is the structure for the dihedral angles for each residue
type Dihedral struct {
	Phi float32 `json:"phi"`
	Psi float32 `json:"psi"`
}

//LoadDihedral parses a UTF8 byte array into one chain
// Size (bytes) | Description
// --------------------------
//          02  | Phi: a 4 byte representation of the Phi angle
//          02  | Psi: a 4 byte representation of the Psi angle
//
// The idea here is that since we're only storing angles to 1/10th of a degree, there are only 3600 values.
func (db *ProteinDB) LoadDihedral(data []byte) error {
	if len(data) == DihedralSize {
		dih := new(Dihedral)
		signAdj := float32(3600)

		switch db.Header.EndianType {
		case "Little":
			temp := make([]byte, 4)
			copy(temp, data[0:2])
			tenths := uint32(binary.LittleEndian.Uint16(temp))
			dih.Phi = (float32(tenths) - signAdj) / 10.0
			copy(temp, data[2:4])
			tenths = uint32(binary.LittleEndian.Uint16(temp))
			dih.Psi = (float32(tenths) - signAdj) / 10.0
		case "Big":
			temp := make([]byte, 4)
			copy(temp, data[0:2])
			tenths := uint32(binary.BigEndian.Uint16(temp))
			dih.Phi = (float32(tenths) - signAdj) / 10.0
			copy(temp, data[2:4])
			tenths = uint32(binary.BigEndian.Uint16(temp))
			dih.Psi = (float32(tenths) - signAdj) / 10.0
		default:
			return fmt.Errorf("Unknown Endienness detected when building manifest: %s", db.Header.EndianType)
		}

		db.Dihedrals = append(db.Dihedrals, *dih)
	} else {
		return fmt.Errorf("%d bytes passed in. Expected %d", len(data), DihedralSize)
	}

	return nil
}

//ProteinDB is the structure for the entire database and other things we need globally
type ProteinDB struct {
	Header       HeaderRec                  `json:"header"`
	Manifest     []ManifestEntry            `json:"manifest"`
	JumpTable    map[JumpKey]JumpTableEntry `json:"jumpTable"`
	JumpIndexMax uint32                     `json:"-"` // Calculated value used in the query
	PDBData      map[string]PDBRec          `json:"pdbData"`
	Chains       []Chain                    `json:"chains"`
	SetMembers   []uint32                   `json:"setMembers"`
	Dihedrals    []Dihedral                 `json:"dihedrals"`
	Residues     string                     `json:"residues"`
	Redis        redis.Conn                 `json:"-"` //Redis connection is never returned as JSON

	querySem     chan struct{} `json:"-"` //Caps concurrent query execution
	queryTimeout time.Duration `json:"-"` //Per-query runtime cap
}

//NewProteinDB creates the ProteinDB structure from a file
func NewProteinDB(fileName string) (*ProteinDB, error) {
	db := new(ProteinDB)
	db.JumpIndexMax = 0

	//Bound concurrent query execution and cap per-query runtime so a shared
	//server can't be exhausted by a flood of queries or a single runaway one.
	//Both are configurable via env vars.
	maxQueries := 8
	if v, err := strconv.Atoi(os.Getenv("DIREDB_MAX_QUERIES")); err == nil && v > 0 {
		maxQueries = v
	}
	db.querySem = make(chan struct{}, maxQueries)
	db.queryTimeout = 120 * time.Second
	if v, err := strconv.Atoi(os.Getenv("DIREDB_QUERY_TIMEOUT")); err == nil && v > 0 {
		db.queryTimeout = time.Duration(v) * time.Second
	}

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Fatal(err)
		return db, err
	}

	log.Printf("Creating a protein database from %s\n", absFileName)
	//open the file
	file, err := os.Open(absFileName)
	if err != nil {
		log.Fatal(err)
		return db, err
	}

	//read bytes for the header
	headerBytes := make([]byte, HeaderSize)
	numBytes, err := file.Read(headerBytes)
	if err != nil {
		log.Fatal(err)
		return db, err
	}

	//check to make sure we read the correct number of bytes
	if numBytes != HeaderSize {
		log.Fatal(fmt.Errorf("%d bytes expected (header), %d bytes read", HeaderSize, numBytes))
		return db, fmt.Errorf("Header: Only read %d bytes", numBytes)
	}

	log.Printf("Parsing and Creating the Header ...\n")
	//load data into the header
	db.LoadHeader(headerBytes)

	log.Printf("Parsing and Creating the Manifest ...\n")
	//allocate space for the manifest
	db.Manifest = make([]ManifestEntry, 0)

	//go to where the manifest starts
	_, err = file.Seek(int64(db.Header.ManifestStart), 0)
	if err != nil {
		log.Fatal(err)
		return db, err
	}

	//read bytes for the all the manifest entries
	manifestBytes := make([]byte, ManifestEntrySize*int(db.Header.ManifestEntries))
	numBytes, err = file.Read(manifestBytes)
	if err != nil {
		log.Fatal(err)
		return db, err
	}

	//check to make sure we read the correct number of bytes
	if numBytes != ManifestEntrySize*int(db.Header.ManifestEntries) {
		log.Fatal(fmt.Errorf("%d bytes expected (manifest entry), %d bytes read", ManifestEntrySize, numBytes))
		return db, fmt.Errorf("Manifest: Only read %d bytes", numBytes)
	}

	for i := 0; i < int(db.Header.ManifestEntries); i++ {
		//load the manafest entry
		db.LoadManifestEntry(manifestBytes[i*ManifestEntrySize : (i+1)*ManifestEntrySize])
	}

	log.Printf("Loading sections from the Manifest ...\n")
	//load all of the other sections "dynamically"
	for _, entry := range db.Manifest {
		log.Printf("Loading the %s - %d entries...\n", entry.Label, entry.NumEntries)

		//go to the starting position for the section
		_, err = file.Seek(int64(entry.Start), 0)
		if err != nil {
			log.Fatal(err)
			return db, err
		}

		//read in the bytes all at once
		dataBytes := make([]byte, uint32(entry.Size)*entry.NumEntries)

		numBytes, err := file.Read(dataBytes)
		if numBytes != int(entry.Size)*int(entry.NumEntries) {
			log.Fatal(fmt.Errorf("%d bytes expected (%s), %d bytes read", int(entry.Size)*int(entry.NumEntries), entry.Label, numBytes))
			return db, fmt.Errorf("%s: Only read %d bytes", entry.Label, numBytes)
		}

		if err != nil {
			log.Fatal(err)
			return db, err
		}

		//load the data chunk by chunk
		switch entry.SectionID {
		case 0:
			db.JumpTable = make(map[JumpKey]JumpTableEntry, entry.NumEntries)
			for i := 0; i < int(entry.NumEntries); i++ {
				db.LoadJumpTableEntry(dataBytes[i*int(entry.Size) : (i+1)*int(entry.Size)])
			}
		case 1:
			db.SetMembers = make([]uint32, 0)
			for i := 0; i < int(entry.NumEntries); i++ {
				db.LoadMember(dataBytes[i*int(entry.Size) : (i+1)*int(entry.Size)])
			}
		case 2:
			db.PDBData = make(map[string]PDBRec, entry.NumEntries)
			for i := 0; i < int(entry.NumEntries); i++ {
				db.LoadPDBEntry(dataBytes[i*int(entry.Size) : (i+1)*int(entry.Size)])
			}
		case 3:
			db.Chains = make([]Chain, 0)
			for i := 0; i < int(entry.NumEntries); i++ {
				db.LoadChain(dataBytes[i*int(entry.Size) : (i+1)*int(entry.Size)])
			}
		case 4:
			db.Dihedrals = make([]Dihedral, 0)
			for i := 0; i < int(entry.NumEntries); i++ {
				db.LoadDihedral(dataBytes[i*int(entry.Size) : (i+1)*int(entry.Size)])
			}
		case 5:
			db.Residues = string(dataBytes)
		default:
			return db, fmt.Errorf("Unknown Section: %d %s", entry.SectionID, entry.Label)
		}
	}

	file.Close()

	//The query path's intersection is a linear sorted-merge, which depends on
	//each set being stored ascending-sorted. Verify that invariant once here so
	//a database that violates it fails loudly at load instead of silently
	//returning incomplete query results.
	if err := db.verifySetMembersSorted(); err != nil {
		log.Fatal(err)
		return db, err
	}

	return db, nil
}

//verifySetMembersSorted checks that every jump-table entry's set of chain
//indices is stored ascending-sorted (the invariant FindIntersection relies on).
//Returns an error on the first violation or out-of-range reference.
func (db *ProteinDB) verifySetMembersSorted() error {
	n := len(db.SetMembers)
	for key, jte := range db.JumpTable {
		start := int(jte.SetStartIndex)
		end := start + int(jte.SetNumMembers)
		if start < 0 || start > end || end > n {
			return fmt.Errorf("jump entry %v references out-of-range set [%d:%d] (set members len %d)", key, start, end, n)
		}
		for i := start + 1; i < end; i++ {
			if db.SetMembers[i] < db.SetMembers[i-1] {
				return fmt.Errorf("set members not sorted for jump entry %v at index %d (%d < %d); the query intersection requires ascending-sorted sets", key, i, db.SetMembers[i], db.SetMembers[i-1])
			}
		}
	}
	log.Printf("Verified set-member sort invariant across %d jump-table entries\n", len(db.JumpTable))
	return nil
}

//DatabaseRoutes is the Routes for Proteins
func (db *ProteinDB) DatabaseRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/header", db.GetHeader)
	router.Get("/manifest", db.GetManifest)
	router.Get("/manifest/{index}", db.GetManifestByIndex)
	router.Get("/jumptable", db.GetJumpTable)
	router.Get("/jumptable/keys", db.GetJumpTableKeys)
	router.Get("/jumptable/residue/{residue}/index/{index}", db.GetJumpTableEntry)
	router.Get("/pdbdata", db.GetPDBData)
	router.Get("/pdbdata/{proteinID}", db.GetPDBDataByID)
	router.Get("/chain", db.GetChains)
	router.Get("/chain/{index}", db.GetChainByIndex)
	router.Get("/setmember", db.GetSetMembers)
	router.Get("/setmember/{index}", db.GetSetMemberByIndex)
	router.Get("/setmember/{index}/length/{length}", db.GetSetMemberArrayByIndex)
	router.Get("/dihedral", db.GetDihedrals)
	router.Get("/dihedral/{index}", db.GetDihedralByIndex)
	router.Get("/residue", db.GetResidues)
	router.Get("/residue/{index}", db.GetResidueByIndex)
	router.Get("/residue/{index}/length/{length}", db.GetResidueStringByIndex)

	return router
}

//FieldSummary is a summary to return for large lists
type FieldSummary struct {
	Name   string   `json:"recordType"`
	Fields []string `json:"fieldNames,omitempty"`
	Length int      `json:"length"`
}

//GetHeader returns the Header in json format
func (db *ProteinDB) GetHeader(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, db.Header)
}

//GetManifest returns the Manifest in json format
func (db *ProteinDB) GetManifest(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, db.Manifest)
}

//GetManifestByIndex returns the Manifest at an index in json format
func (db *ProteinDB) GetManifestByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.Manifest) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, db.Manifest[index])
}

//GetJumpTable returns the Jump Table in json format
func (db *ProteinDB) GetJumpTable(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "JumpTableEntry"
	s.Fields = []string{"residueName", "padding", "resIndex", "setNumMembers", "setStartIndex"}
	s.Length = len(db.JumpTable)

	render.JSON(w, r, s)
}

//GetJumpTableKeys returns the keys for the Jump Table in json format
func (db *ProteinDB) GetJumpTableKeys(w http.ResponseWriter, r *http.Request) {
	keys := make([]JumpKey, len(db.JumpTable))
	values := make([]JumpTableEntry, len(db.JumpTable))

	i := 0
	for k, v := range db.JumpTable {
		keys[i] = k
		values[i] = v
		i++
	}

	render.JSON(w, r, keys)
}

//GetJumpTableEntry returns the keys for the Jump Table in json format
func (db *ProteinDB) GetJumpTableEntry(w http.ResponseWriter, r *http.Request) {
	resName := strings.ToUpper(chi.URLParam(r, "residue"))
	resIndex, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	key := new(JumpKey)
	key.ResidueName = resName
	key.ResIndex = uint32(resIndex)

	jte, found := db.JumpTable[*key]
	if !found {
		err = fmt.Errorf("Record %v was not found", key)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, jte)
}

//GetPDBData returns the PDB Data in json format
func (db *ProteinDB) GetPDBData(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "PDBRec"
	s.Fields = []string{"name", "numChains", "chainIndex"}
	s.Length = len(db.PDBData)

	render.JSON(w, r, s)
}

//GetPDBDataByID returns the PDB Data in json format
func (db *ProteinDB) GetPDBDataByID(w http.ResponseWriter, r *http.Request) {
	pdbName := strings.ToUpper(chi.URLParam(r, "proteinID"))

	pdb, found := db.PDBData[pdbName]
	if !found {
		err := fmt.Errorf("Record for %s was not found", pdbName)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, pdb)
}

//GetChains returns the Chain Data in json format
func (db *ProteinDB) GetChains(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "Chain"
	s.Fields = []string{"sourcePDB", "sourceIndex", "chainID", "padding", "numModels", "numResidues", "modelLength", "modelIndex", "residueIndex"}
	s.Length = len(db.Chains)

	render.JSON(w, r, s)
}

//GetChainByIndex returns the Chain Data in json format
func (db *ProteinDB) GetChainByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.Chains) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, db.Chains[index])
}

//GetResidues returns the Residue Data in json format
func (db *ProteinDB) GetResidues(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "Residue"
	s.Fields = []string{}
	s.Length = len(db.Residues)

	render.JSON(w, r, s)
}

//GetResidueByIndex returns the Residue Data in json format
func (db *ProteinDB) GetResidueByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.Residues) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, strings.Split(db.Residues[index:index+1], ""))
}

//GetResidueStringByIndex returns the Residue Data in json format
func (db *ProteinDB) GetResidueStringByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	length, err := strconv.Atoi(chi.URLParam(r, "length"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.Residues) || index+length > len(db.Residues) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, strings.Split(db.Residues[index:index+length], ""))
}

//GetSetMembers returns the Set Member Data in json format
func (db *ProteinDB) GetSetMembers(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "SetMember"
	s.Fields = []string{}
	s.Length = len(db.SetMembers)

	render.JSON(w, r, s)
}

//GetSetMemberByIndex returns a specificSet Member Data in json format
func (db *ProteinDB) GetSetMemberByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.SetMembers) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, db.SetMembers[index])
}

//GetSetMemberArrayByIndex returns a specificSet Member Data in json format
func (db *ProteinDB) GetSetMemberArrayByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	length, err := strconv.Atoi(chi.URLParam(r, "length"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.SetMembers) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, db.SetMembers[index:index+length])
}

//GetDihedrals returns the Dihedral Data in json format
func (db *ProteinDB) GetDihedrals(w http.ResponseWriter, r *http.Request) {
	s := new(FieldSummary)
	//TODO: Get these values from reflection instead of hard coding
	s.Name = "Dihedral"
	s.Fields = []string{"phi", "psi"}
	s.Length = len(db.Dihedrals)

	render.JSON(w, r, s)
}

//GetDihedralByIndex returns the Dihedral Data in json format
func (db *ProteinDB) GetDihedralByIndex(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(chi.URLParam(r, "index"))

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if index < 0 || index >= len(db.Dihedrals) {
		err := fmt.Errorf("Index %d is not in range", index)
		render.Render(w, r, ErrNotFound(err))
		return
	}

	render.JSON(w, r, db.Dihedrals[index])
}
