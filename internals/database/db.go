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

// NewDB Create a new database connection
func NewDB(path string) (*DB, error) {
	mux := &sync.RWMutex{}
	database := DB{
		path: path,
		mux:  mux,
	}

	err := database.ensureDB()
	// TODO: Not sure if I should return nil, err or not
	if err != nil {
		return nil, err
	}

	return &database, nil
}

// CreateChirp creates a new chirp and saves it to disk
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

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
}

func (db *DB) loadDB() (DBStructure, error) {
	db.mux.Lock()

	dbStructure := DBStructure{
		Chirps: map[int]Chirp{},
	}

	file, err := os.OpenFile(db.path, os.O_RDONLY, 0755)
	if err != nil {
		return dbStructure, err
	}
	fileData, err := os.ReadFile(db.path)
	if err != nil {
		return dbStructure, err
	}

	err = json.Unmarshal(fileData, &dbStructure)
	if err != nil {
		return dbStructure, err
	}

	err = file.Close()
	if err != nil {
		return dbStructure, err
	}

	defer db.mux.Unlock()
	return dbStructure, err
}

func (db *DB) ensureDB() error {
	_, err := os.Stat(db.path)
	if err != nil {
		// create db
		file, err := os.Create(db.path)
		if err != nil {
			return err
		}

		return file.Close()
	}

	return nil
}
