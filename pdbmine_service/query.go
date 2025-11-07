package main

import (
	"archive/tar"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	//Chi Router
	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	//UUID
	"github.com/google/uuid"
)

//QueryStatus is the status of a query that's running
type QueryStatus int

const (
	//Error is a query that failed or is in an error state
	Error QueryStatus = 0
	//Running is a query that is currently executing
	Running QueryStatus = 1
	//Complete is a query that has been finished
	Complete QueryStatus = 2
)

//String returns a string representation of a QueryStatus
func (status QueryStatus) String() string {
	names := [...]string{
		"Error",
		"Running",
		"Complete"}

	if status < Error || status > Complete {
		return "Unknown"
	}

	return names[status]
}

//ParseQueryStatus a String to determine the QueryStatus
func ParseQueryStatus(text string) (QueryStatus, error) {
	if text == "Error" {
		return Error, nil
	}

	if text == "Running" {
		return Running, nil
	}

	if text == "Complete" {
		return Complete, nil
	}

	return Error, errors.New("unknown status")
}

//QueryRoutes is the Routes for Queries
func (db *ProteinDB) QueryRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", db.CreateQuery)
	router.Get("/{queryID}", db.GetQueryResultsJSON)
	router.Get("/{queryID}/tarball", db.GetQueryResultsFile)

	return router
}

//QueryRequest is the incoming request for a query
type QueryRequest struct {
	Residue string `json:"residueChain"`
	Length  int    `json:"codeLength"`
	Window  int    `json:"windowSize"`
}

//Bind binds the request to the QueryRequest with some error checking
func (q *QueryRequest) Bind(r *http.Request) error {
	//Check to make sure some JSON was sent in
	if q == nil {
		return errors.New("incorrect parameters to create a query request")
	}

	//Check to make sure that the window length is less than the total residue length
	if q.Window > len(q.Residue) {
		return errors.New("the window length cannot be larger that the length of the residue chain")
	}

	//Check to make sure that the letter code length is either 1 or 3. Long term we could figure this out
	if q.Length != 1 && q.Length != 3 {
		return errors.New("the amino acid code length must be 1 or 3")
	}

	//If the letter code length is 3, then make sure that residue chain is comprised of complete codes
	if q.Length == 3 && len(q.Residue)%3 != 0 {
		return errors.New("the length of the residue chain does not match 3 character codes")
	}

	return nil
}

//QueryResponse is the response for the complete query
type QueryResponse struct {
	Status  string                                 `json:"status,omitempty"`
	QueryID string                                 `json:"queryID,omitempty"`
	Frames  map[string]map[string][][]AminoAcidRec `json:"frames,omitempty"`
}

//TranslateToOneLetterCode translates from 3 code amino acids to 1 code amino acids since 1 code is what the database hase
func TranslateToOneLetterCode(threeLetter []string) []string {
	//Create a map between the three letter codes and one letter codes
	threeToOne := map[string]string{
		"ALA": "A",
		"ARG": "R",
		"ASN": "N",
		"ASP": "D",
		"ASX": "B",
		"CYS": "C",
		"GLU": "E",
		"GLN": "Q",
		"GLX": "Z",
		"GLY": "G",
		"HIS": "H",
		"ILE": "I",
		"LEU": "L",
		"LYS": "K",
		"MET": "M",
		"PHE": "F",
		"PRO": "P",
		"SER": "S",
		"THR": "T",
		"TRP": "W",
		"TYR": "Y",
		"VAL": "V",
	}

	oneLetter := make([]string, len(threeLetter))

	//Convert the three letter code to a one letter code one by one
	for i, three := range threeLetter {
		one, found := threeToOne[strings.ToUpper(three)]

		//If an unknown three letter code is provided, then default to X
		if !found {
			one = "X"
		}

		oneLetter[i] = one
	}

	return oneLetter
}

//FindIntersection finds the intersection between two sets of integers
func FindIntersection(first []uint32, second []uint32) []uint32 {
	result := make([]uint32, 0)
	//Put the first list into a hash
	firstHash := make(map[uint32]uint8, len(first))
	for _, entry := range first {
		firstHash[entry] = 1
	}

	//check every item of the second list to see if it's in the hash. If it is, then add it to the result
	for _, entry := range second {
		_, found := firstHash[entry]

		if found {
			result = append(result, entry)
		}
	}

	return result
}

//NewQuery performs a new query on a request and returns it
func (db *ProteinDB) NewQuery(query *QueryRequest) (QueryResponse, error) {
	resp := new(QueryResponse)
	resp.Frames = make(map[string]map[string][][]AminoAcidRec)

	//strip out all white space in the Residue string.
	strings.Replace(query.Residue, "\t", "", -1)
	strings.Replace(query.Residue, "\n", "", -1)
	strings.Replace(query.Residue, " ", "", -1)

	//Check the length from the request
	switch query.Length {
	case 1:
		//divide into frames with the length of the window and search for each one.
		for i := 0; i < len(query.Residue)-query.Window+1; i++ {
			frame := query.Residue[i : i+query.Window]
			frame = fmt.Sprintf("%03d_%s", i, strings.ToUpper(frame))
			frameArray := strings.SplitAfter(frame[4:], "") //skip 3 digits and _
			resp.Frames[frame] = db.SearchForFrame(frameArray)
		}
	case 3:
		if 0 == len(query.Residue)%3 {
			threeCode := regexp.MustCompile(".{3}")

			//divide into frames with the length of the window and search for each one.
			for i := 0; i < len(query.Residue)-3*query.Window+1; i += 3 {
				frame := query.Residue[i : i+query.Window*3]
				frame = fmt.Sprintf("%03d_%s", i, strings.ToUpper(frame))
				frameArray := threeCode.FindAllString(frame[4:], -1) //skip 3 digits and _
				//Convert from three letter code to one letter code
				oneFrameArray := TranslateToOneLetterCode(frameArray)
				resp.Frames[frame] = db.SearchForFrame(oneFrameArray)
			}
		} else {
			return *resp, errors.New("residue string is not evenly divisible by 3")
		}
	default:
		return *resp, errors.New("unable to determine the length of each residue name")
	}

	resp.Status = Complete.String()

	return *resp, nil
}

//SearchForFrame searches for frames by comparing sets
func (db *ProteinDB) SearchForFrame(residues []string) map[string][][]AminoAcidRec {
	result := make(map[string][][]AminoAcidRec, 0)

	log.Printf("%v\n", residues)
	//for each starting point to the number of possible residues
	for i := 0; i < int(db.JumpIndexMax); i++ {
		//for each residue
		finalSet := make([]uint32, 0)
		for j, residue := range residues {
			//get the set members by using the jump table
			key := new(JumpKey)
			key.ResidueName = residue
			key.ResIndex = uint32(i + j)
			jte, found := db.JumpTable[*key]

			//if this position is found
			if found {
				startIndex := int(jte.SetStartIndex)
				endIndex := int(jte.SetStartIndex) + int(jte.SetNumMembers)

				if 0 == j {
					finalSet = db.SetMembers[startIndex:endIndex]
				} else {
					set := db.SetMembers[startIndex:endIndex]
					//get intersection between finalSet and set
					finalSet = FindIntersection(finalSet, set)
				}
			} else {
				//start over
				//Make finalSet empty so nothing is saved since the position wasn't found
				finalSet = make([]uint32, 0)
				break
			}
		}

		//i is the starting index inside the chain/model
		for _, chainNum := range finalSet {
			log.Printf("Residue found in %s_%s at index %d; %d models", db.Chains[chainNum].SourcePDB, db.Chains[chainNum].ChainID, i, db.Chains[chainNum].NumModels)

			tag := fmt.Sprintf("%s_%s", db.Chains[chainNum].SourcePDB, db.Chains[chainNum].ChainID)
			for j := 0; j < int(db.Chains[chainNum].NumModels); j++ {
				rIndex := int(db.Chains[chainNum].ResidueIndex) + i
				mIndex := int(db.Chains[chainNum].ModelIndex) + j*int(db.Chains[chainNum].ModelLength) + i

				//Add each amino acid to the string
				resString := make([]AminoAcidRec, len(residues))
				for k := 0; k < len(residues); k++ {
					aa := new(AminoAcidRec)
					aa.Residue = db.Residues[rIndex : rIndex+1]
					aa.Phi = db.Dihedrals[mIndex].Phi
					aa.Psi = db.Dihedrals[mIndex].Psi
					rIndex++
					mIndex++
					resString[k] = *aa
				}

				result[tag] = append(result[tag], resString)
			}
		}
	}

	return result
}

//CreateQuery initiates a query for a given fragment and window size
func (db *ProteinDB) CreateQuery(w http.ResponseWriter, r *http.Request) {
	//Build and execute the query.

	//Check the incoming request
	post := &QueryRequest{}
	render.SetContentType(render.ContentTypeJSON)
	if err := render.Bind(r, post); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	//log.Printf("%+v", post)

	resp := new(QueryResponse)

	//Get a UUID to save results and status
	queryID, err := uuid.NewUUID()
	//If I can't get a UUID, fail since there's no key and no way of storing results.
	if err != nil {
		render.Render(w, r, ErrCannotGenerateUID(err))
		return
	}

	//Check for a saved query to see if this even needs to be run
	savedID, savedStatus, err := db.searchForSavedQuery(post)

	//If no error, then we found the record
	if err == nil {
		log.Println("Saved record found for query")
		resp.QueryID = savedID
		resp.Status = savedStatus
	} else {
		log.Printf("Creating new query: %s\n", queryID.String())
		//We need to run the whole query
		resp.QueryID = queryID.String()
		resp.Status = Running.String()

		//try and save the query for next time. If we fail just go on
		db.SaveQuery(queryID.String(), post)

		//Report the query has been created
		render.Status(r, http.StatusCreated)

		go db.RunAndSaveResults(post, resp.QueryID)
	}

	render.JSON(w, r, resp)
}

//GetQueryResultsJSON returns the results of a query in JSON format
func (db *ProteinDB) GetQueryResultsJSON(w http.ResponseWriter, r *http.Request) {
	queryID := chi.URLParam(r, "queryID")
	queryParam := r.URL.Query() //filters

	showStatus := false
	showResults := false
	showQueryID := false

	//If there's a filter on what to show, then only show part of it.
	if len(queryParam["show"]) != 0 {
		for i := range queryParam["show"] {
			switch name := queryParam["show"][i]; strings.ToLower(name) {
			case "status":
				showStatus = true
			case "queryid":
				showQueryID = true
			case "frames":
				showResults = true
			}
		}
	} else {
		showStatus = true
		showResults = true
		showQueryID = true
	}

	resp, err := db.RetrieveJSON(queryID)
	if err != nil {
		render.Render(w, r, ErrNotFound(err))
		return
	}

	if !showStatus {
		resp.Status = ""
	}

	if !showResults {
		resp.Frames = nil
	}

	if !showQueryID {
		resp.QueryID = ""
	}

	render.JSON(w, r, resp)
}

//GetQueryResultsFile returns the results of a query as a Tarball
func (db *ProteinDB) GetQueryResultsFile(w http.ResponseWriter, r *http.Request) {
	queryID := chi.URLParam(r, "queryID")

	//First, see if the file is there. If so, just return it
	name := queryID + ".tar"
	tarball, err := ReadFromDisk(name)

	//If the tarball wasn't there or we couldn't read it, try to generate from JSON
	if err != nil {
		resp, err := db.RetrieveJSON(queryID)
		tarball, err = writeQueryTarball(resp)

		if err != nil {
			render.Render(w, r, ErrNotFound(err))
			return
		}
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	//w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(tarball)), 10))
	io.Copy(w, bytes.NewReader(tarball))
	return
}

func writeQueryTarball(record *QueryResponse) ([]byte, error) {
	buff := &bytes.Buffer{} // creates IO Writer for complete tarball
	tw := tar.NewWriter(buff)
	resSummary := make(map[string][][]string)
	defer tw.Close()

	//create a directory in the tarball for each frame
	for frame := range record.Frames {
		sumBuff := &bytes.Buffer{} // creates IO Writer for summary file
		w := csv.NewWriter(sumBuff)

		for protein := range record.Frames[frame] {
			//create a csv file for each model in a protein
			for model := range record.Frames[frame][protein] {
				//generate a summary line for the frame
				summaryLine := generateSummaryLine(protein[0:4], protein[5:6], fmt.Sprintf("%02d", model), record.Frames[frame][protein][model])

				if err := w.Write(summaryLine); err != nil {
					log.Fatalln("error writing record to csv:", err)
					return nil, err
				}

				//add summary lines to each residue in the frame
				//frame is of the format ###_CCCC, split this up
				parts := strings.Split(frame, "_")
				frameArray := strings.SplitAfter(parts[1], "")
				frameIndex, err := strconv.Atoi(parts[0])
				if err != nil {
					log.Fatalf("Expected a id to be a number. Instead received %s", parts[0])
					return nil, err
				}

				//for each residue in the frame, add a summary line
				for offset, residue := range frameArray {
					//key is the frame seq + the offset into the frame
					resSumKey := fmt.Sprintf("%03d_%s", frameIndex+offset, residue)

					//init the slice if one isn't there
					if _, exists := resSummary[resSumKey]; !exists {
						resSummary[resSumKey] = make([][]string, 0)
					}

					//add summary line to slice
					resLine := generateResSummaryLine(protein[0:4], protein[5:6], fmt.Sprintf("%02d", model), record.Frames[frame][protein][model], offset)
					resSummary[resSumKey] = append(resSummary[resSumKey], resLine)
				}

				//create the model csv file
				content, _ := writeModelCSV(record.Frames[frame][protein][model])

				hdr := &tar.Header{
					Name: fmt.Sprintf("%s%s%s%s%s_%02d.csv", "fragment_data", string(filepath.Separator), frame, string(filepath.Separator), protein, model),
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

		//flush the summary file
		w.Flush()

		//write the summary file
		hdr := &tar.Header{
			Name: fmt.Sprintf("%s%s%s_summary.csv", "fragment_data", string(filepath.Separator), frame),
			Mode: 0600,
			Size: int64(len(sumBuff.Bytes())),
		}

		//write the summary csv tar header
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		//write the summary data
		if _, err := tw.Write(sumBuff.Bytes()); err != nil {
			return nil, err
		}
	}

	//add the residue summaries in the tarball
	//for each residue
	for residue, lines := range resSummary {
		//create a buffer for the summary file
		content, _ := writeResSummaryCSV(lines)

		hdr := &tar.Header{
			Name: fmt.Sprintf("%s%s%s.csv", "residue_data", string(filepath.Separator), residue),
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
	return buff.Bytes(), nil
}

func generateSummaryLine(protein string, chainID string, model string, chain []AminoAcidRec) []string {
	line := []string{
		protein, chainID, model,
	}

	var aminoAcids []string
	var angles []string

	for i := range chain {
		aminoAcids = append(aminoAcids, chain[i].Residue)
		angles = append(angles, fmt.Sprintf("%.1f", chain[i].Phi))
		angles = append(angles, fmt.Sprintf("%.1f", chain[i].Psi))
	}

	line = append(line, aminoAcids...)
	line = append(line, angles...)

	return line
}

func generateResSummaryLine(protein string, chainID string, model string, chain []AminoAcidRec, offset int) []string {
	line := []string{
		protein, chainID, model,
	}

	var aminoAcids []string

	for i := range chain {
		aminoAcids = append(aminoAcids, chain[i].Residue)
	}

	line = append(line, aminoAcids...)
	line = append(line, fmt.Sprintf("%.1f", chain[offset].Phi))
	line = append(line, fmt.Sprintf("%.1f", chain[offset].Psi))

	return line
}

func writeResSummaryCSV(lines [][]string) ([]byte, error) {
	buff := &bytes.Buffer{} // creates IO Writer
	w := csv.NewWriter(buff)

	for _, line := range lines {
		if err := w.Write(line); err != nil {
			log.Fatalln("error writing record to csv:", err)
			return nil, err
		}
	}

	w.Flush()

	return buff.Bytes(), nil
}
