package main

import (
	"archive/tar"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	//Chi Router
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

//AminoAcidRec is a JSON representation of an amino acid
type AminoAcidRec struct {
	Residue string  `json:"residueName"`
	Phi     float32 `json:"phi"`
	Psi     float32 `json:"psi"`
}

//ProteinModel is the representation of a model in a chain
//type ProteinModel struct {
//	Residues []AminoAcidRec `json:"residues"`
//}

//ProteinRec is a JSON representation of a protein
type ProteinRec struct {
	Name   string                      `json:"proteinName"`
	Chains map[string][][]AminoAcidRec `json:"chains" xml:"AminoAcidRec>AminoAcid"`
}

//NewProteinRec create a Protein Record to return given an ID
func NewProteinRec(db *ProteinDB, pdbName string) (ProteinRec, error) {
	record := new(ProteinRec)
	pdbName = strings.ToUpper(pdbName)

	log.Printf("Fetching %s", pdbName)

	pdb, found := db.PDBData[pdbName]

	//Check to make sure the protein is found in the database
	if found {
		record.Name = strings.ToUpper(pdbName)
		record.Chains = make(map[string][][]AminoAcidRec, pdb.NumChains)
		//record.Chains = make(map[string][]ProteinModel, pdb.NumChains)
		log.Printf("%d Chains", pdb.NumChains)

		cIndex := int(pdb.ChainIndex)
		//TODO: Add check to insure that ModelLength == ResidueLength
		//For each chain, add all of the models
		for i := 0; i < int(pdb.NumChains); i++ {
			log.Printf("%d models in Chain %s", db.Chains[cIndex+i].NumModels, db.Chains[cIndex+i].ChainID)
			log.Printf("%d residues in each model", db.Chains[cIndex+i].NumResidues)
			//Initialize the slice of residues
			record.Chains[db.Chains[cIndex+i].ChainID] = make([][]AminoAcidRec, int(db.Chains[cIndex+i].NumModels))
			//record.Chains[db.Chains[cIndex+i].ChainID] = make([]ProteinModel, int(db.Chains[cIndex+i].NumModels))
			//model := new(ProteinModel)

			//Add each model to the chain
			for j := 0; j < int(db.Chains[cIndex+i].NumModels); j++ {

				rIndex := db.Chains[cIndex+i].ResidueIndex
				mIndex := int(db.Chains[cIndex+i].ModelIndex) + j*int(db.Chains[cIndex+i].ModelLength)

				//Add each amino acid to the string
				residues := make([]AminoAcidRec, db.Chains[cIndex+i].NumResidues)
				for k := 0; k < int(db.Chains[cIndex+i].NumResidues); k++ {
					aa := new(AminoAcidRec)
					aa.Residue = db.Residues[rIndex : rIndex+1]
					aa.Phi = db.Dihedrals[mIndex].Phi
					aa.Psi = db.Dihedrals[mIndex].Psi
					rIndex++
					mIndex++
					residues[k] = *aa
				}

				record.Chains[db.Chains[cIndex+i].ChainID][j] = residues
				//model.Residues = residues
				//record.Chains[db.Chains[cIndex+i].ChainID][j] = *model
			}
		}
	} else {
		return ProteinRec{}, fmt.Errorf("%s not found", pdbName)
	}

	return *record, nil
}

//ProteinRoutes is the Routes for Proteins
func (db *ProteinDB) ProteinRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", db.GetAllProteins)
	router.Get("/{proteinID}", db.GetProteinJSON)
	router.Get("/{proteinID}/tarball", db.GetProteinFile)

	return router
}

//GetAllProteins returns a list of all of the Protein ID's available in the database
func (db *ProteinDB) GetAllProteins(w http.ResponseWriter, r *http.Request) {
	keys := make([]string, len(db.PDBData))

	i := 0
	for k := range db.PDBData {
		keys[i] = k
		i++
	}

	render.JSON(w, r, keys)
}

//GetProteinJSON returns the amino acid residues and dihedral angles for a given protein in JSON format
func (db *ProteinDB) GetProteinJSON(w http.ResponseWriter, r *http.Request) {
	db.getProtein(w, r, false)
}

//GetProteinFile returns the amino acid residues and dihedral angles for a given protein in a tarball
func (db *ProteinDB) GetProteinFile(w http.ResponseWriter, r *http.Request) {
	db.getProtein(w, r, true)
}

//getProtein returns the amino acid residues and dihedral angles for a given protein
func (db *ProteinDB) getProtein(w http.ResponseWriter, r *http.Request, retFile bool) {
	pdbName := chi.URLParam(r, "proteinID")
	query := r.URL.Query()
	chain := query["chain"] //Get query parameters for chain

	fullRecord, err := NewProteinRec(db, pdbName)

	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}

	record := new(ProteinRec)
	record.Name = fullRecord.Name

	//if no chain is specified, return the whole chain.
	if len(chain) == 0 {
		record = &fullRecord
	} else {
		//if we have chains, only return the chains listed
		record.Chains = make(map[string][][]AminoAcidRec, len(chain))
		for _, c := range chain {
			c = strings.ToUpper(c)
			_, found := fullRecord.Chains[c]
			if found {
				record.Chains[c] = fullRecord.Chains[c]
			}
		}
	}

	if retFile {
		tarball, _ := writeProteinTarball(*record)

		//Send the headers
		Filename := fmt.Sprintf("%s.tar", record.Name)
		w.Header().Set("Content-Disposition", "attachment; filename="+Filename)
		w.Header().Set("Content-Type", "application/tar")
		w.Header().Set("Content-Length", strconv.FormatInt(int64(len(tarball)), 10))
		io.Copy(w, bytes.NewReader(tarball))
	} else {
		render.JSON(w, r, record)
	}
}

func writeProteinTarball(record ProteinRec) ([]byte, error) {
	buff := &bytes.Buffer{} // creates IO Writer
	//gz := gzip.NewWriter(buff)
	//defer gz.Close()
	//tw := tar.NewWriter(gz)
	tw := tar.NewWriter(buff)
	defer tw.Close()

	for chain := range record.Chains {
		for model := range record.Chains[chain] {
			content, _ := writeModelCSV(record.Chains[chain][model])

			hdr := &tar.Header{
				Name: fmt.Sprintf("%s_%s_%02d.csv", record.Name, chain, model),
				Mode: 0600,
				Size: int64(len(content)),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return nil, err
			}

			if _, err := tw.Write([]byte(content)); err != nil {
				return nil, err
			}

		}
	}

	return buff.Bytes(), nil
}

func writeModelCSV(chain []AminoAcidRec) ([]byte, error) {
	buff := &bytes.Buffer{} // creates IO Writer
	w := csv.NewWriter(buff)

	line := []string{
		"Residue", "Phi", "Psi",
	}
	if err := w.Write(line); err != nil {
		log.Fatalln("error writing record to csv:", err)
		return nil, err
	}

	for i := range chain {
		line := []string{
			chain[i].Residue,
			fmt.Sprintf("%.1f", chain[i].Phi),
			fmt.Sprintf("%.1f", chain[i].Psi),
		}
		if err := w.Write(line); err != nil {
			log.Fatalln("error writing record to csv:", err)
			return nil, err
		}
	}

	w.Flush()

	return buff.Bytes(), nil
}
