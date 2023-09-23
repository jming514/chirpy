package database

import (
	"encoding/json"
	"fmt"
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
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	mux := &sync.RWMutex{}
	database := DB{
		path: path,
		mux:  mux,
	}

	// creates the database file if it doesn't exist
	err := database.ensureDB()
	if err != nil {
		return nil, err
	}

	return &database, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	size := len(dbStructure.Chirps)

	newChirp := Chirp{
		Id:   size + 1,
		Body: body,
	}

	dbStructure.Chirps[newChirp.Id] = newChirp

	err = db.writeDB(dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	return newChirp, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	var respSlice []Chirp

	for _, v := range dbStructure.Chirps {
		respSlice = append(respSlice, v)
	}

	return respSlice, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
	_, err := os.Stat(db.path)
	if err != nil {
		fmt.Println("create file...")
		// create db
		file, err := os.Create(db.path)
		if err != nil {
			return err
		}

		_, err = file.Write([]byte("{}"))
		if err != nil {
			return err
		}

		db.mux.Unlock()
		return file.Close()
	}

	db.mux.Unlock()
	return nil
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()

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

	defer db.mux.RUnlock()
	return dbStructure, err
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	newFileData, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(db.path, os.O_RDWR, 0755)
	if err != nil {
		return err
	}

	_, err = file.Write(newFileData)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	db.mux.Unlock()
	return nil
}
