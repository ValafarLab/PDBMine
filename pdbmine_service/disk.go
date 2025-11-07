package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const (
	//DefaultDirectory os the default directory that the data gets saved to
	DefaultDirectory string = "./data"
	//FileMode is the mode to use when creating file
	FileMode os.FileMode = 0644
	//DirMode is the moder to use when creating directory
	DirMode os.FileMode = 0777
)

//SaveToDisk saves a string with json into a file
func SaveToDisk(name string, json []byte) error {
	//Get the directory to write to
	directory := os.Getenv("RESULT_DIR")
	if directory == "" {
		directory = DefaultDirectory
		log.Printf("RESULT_DIR is not set. Defaulting to %s", directory)
	}

	fileName := filepath.Join(directory, name)

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		os.Mkdir(directory, DirMode)
	}

	err := ioutil.WriteFile(fileName, json, FileMode)

	return err
}

//ReadFromDisk reads a json file with the results in from disk
func ReadFromDisk(name string) ([]byte, error) {
	//Get the directory to write to
	directory := os.Getenv("RESULT_DIR")
	if directory == "" {
		directory = DefaultDirectory
		log.Printf("RESULT_DIR is not set. Defaulting to %s", directory)
	}

	fileName := filepath.Join(directory, name)

	data, err := ioutil.ReadFile(fileName)

	return data, err
}

//RunAndSaveResults runs the query and saves the result to disk
func (db *ProteinDB) RunAndSaveResults(post *QueryRequest, queryID string) {
	resp, err := db.NewQuery(post)

	log.Printf("query %s completed", queryID)

	resp.QueryID = queryID //Set the query ID

	if err != nil {
		log.Println("Error running query", err)
		return
	}

	stringResp, err := json.Marshal(resp)
	if err != nil {
		log.Println("Error converting to String", err)
		_, err = db.Redis.Do("HMSET", queryID, "status", Error)
		return
	}

	log.Printf("Results string is %d bytes long", len(stringResp))

	//Try to save to Redis if that's an option.
	writeToDisk := true
	if db.RedisAvailable() {
		writeToDisk = false
		err = db.SaveToRedis(queryID, stringResp)
		if err != nil {
			//Saving to Redis failed so save it to the disk
			writeToDisk = true
		}
	}

	//If there was an error try and save it to the disk, otherwise the results are in Redis
	if writeToDisk {
		log.Println("Saving results to disk")
		//Save the json to disk
		if err != nil {
			log.Println("Error converting results to JSON for file save", err)
			return
		}

		name := queryID + ".json"
		err = SaveToDisk(name, stringResp)

		if err != nil {
			log.Printf("Error saving %s to disk", name)
			log.Println(err)
			return
		}
	}

	//Save the tarball to disk no matter what for now.
	name := queryID + ".tar"
	tarball, err := writeQueryTarball(&resp)
	if err != nil {
		log.Println("Error creating tar file", err)
		return
	}

	err = SaveToDisk(name, tarball)

	if err != nil {
		log.Printf("Error saving %s to disk", name)
		log.Println(err)
		return
	}
}

//searchForSavedQuery looks for a query that's already been run and returns the ID
func (db *ProteinDB) searchForSavedQuery(post *QueryRequest) (string, string, error) {
	//check to see if it's in Redis
	if db.RedisAvailable() {
		log.Println("Redis available. Searching Redis.")
		//Check to see if the query has already been made.
		savedID, err := db.GetQueryFromRedis(*post)
		if err != nil {
			log.Println("Error checking Redis for saved query")
			return "", "", errors.New("unable to check Redis")
		}

		//if this query has already been run, just look it up and return
		if savedID != "" {
			log.Printf("Matching query found with queryID: %s\n", savedID)
			resp, err := db.GetStatusFromRedis(savedID)

			if err != nil {
				log.Println("Error retrieving saved query")
				return "", "", errors.New("unable to retrieve status")
			}

			return savedID, resp.Status, nil
		}
	}

	log.Println("No saved query found")
	//TODO: check to see if it's on Disk
	return "", "", errors.New("unable to find query")
}

//SaveQuery saves a query so that we don't re-run queries. Right now this only really works with Redis.
func (db *ProteinDB) SaveQuery(queryID string, post *QueryRequest) error {
	if db.RedisAvailable() {
		//err := db.SaveQueryInRedis(*QueryRequest, queryID)
		//return err
	}

	return nil
}

//RetrieveJSON tried to load saved JSON data from Redis or from the Disk
func (db *ProteinDB) RetrieveJSON(queryID string) (*QueryResponse, error) {
	resp := new(QueryResponse)
	var err error

	//check to see if it's in Redis
	if db.RedisAvailable() {
		*resp, err = db.GetResponseFromRedis(queryID)
		if err == nil {
			return resp, err
		}
	}

	//If Redis wasn't available or there was an error, then try to read it from disk
	name := queryID + ".json"
	data, err := ReadFromDisk(name)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(data, resp)

	return resp, err
}
