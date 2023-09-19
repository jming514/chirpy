package database

import (
	"encoding/json"
	"os"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

func NewDB(path string) (*DB, error) {
	mux := &sync.RWMutex{}
	database := DB{
		path: path,
		mux:  mux,
	}

	_, err := os.Stat(path)
	if err != nil {
		// create db
		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}

		defer file.Close()
	}

	return &database, nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mux.Lock()
	file, err := os.OpenFile(db.path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return Chirp{}, err
	}

	fileData, err := os.ReadFile(db.path)
	if err != nil {
		return Chirp{}, err
	}

	dbStructure := DBStructure{
		Chirps: map[int]Chirp{},
	}

	err = json.Unmarshal(fileData, &dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	size := len(dbStructure.Chirps)

	newChirp := Chirp{
		Id:   size + 1,
		Body: body,
	}

	dbStructure.Chirps[newChirp.Id] = newChirp

	newFileData, err := json.Marshal(dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	_, err = file.Write(newFileData)
	if err != nil {
		return Chirp{}, err
	}

	defer file.Close()
	defer db.mux.Unlock()
	return newChirp, nil
}
