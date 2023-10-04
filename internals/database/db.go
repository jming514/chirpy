package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type DB struct {
	mux  *sync.RWMutex
	path string
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
	Tokens map[int]Token `json:"tokens"`
}

type Token struct {
	Id         string `json:"id"`
	RevokeTime string `json:"revokeTime"`
}

type User struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Id       int    `json:"id"`
}

type UserReturn struct {
	Email         string `json:"email"`
	Password      string `json:"-"`
	Token         string `json:"token,omitempty"`
	Refresh_Token string `json:"refresh_token,omitempty"`
	Id            int    `json:"id"`
}

type Chirp struct {
	Body string `json:"body"`
	Id   int    `json:"id"`
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

// Login checks if user exists with password, and if so, returns the user
func (db *DB) Login(email, password string) (UserReturn, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return UserReturn{}, err
	}

	for _, value := range dbStructure.Users {
		if value.Email == email && value.Password == password {
			return UserReturn{
				Id:    value.Id,
				Email: value.Email,
			}, nil
		}
	}

	return UserReturn{}, errors.New("user not found")
}

// CreateUser creates a new user and saves it to disk
func (db *DB) CreateUser(email string, password string) (UserReturn, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return UserReturn{}, err
	}

	for _, value := range dbStructure.Users {
		if value.Email == email {
			err := errors.New("user already exists")
			return UserReturn{}, err
		}
	}

	size := len(dbStructure.Users)
	newUser := User{
		Id:       size + 1,
		Email:    email,
		Password: password,
	}
	dbStructure.Users[newUser.Id] = newUser

	err = db.writeDB(dbStructure)
	if err != nil {
		return UserReturn{}, err
	}

	return UserReturn{
		Id:    newUser.Id,
		Email: newUser.Email,
	}, nil
}

func (db *DB) UpdateUser(u User) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for key, value := range dbStructure.Users {
		if value.Id == u.Id {
			// update user
			updatedUser := User{
				Id:       value.Id,
				Email:    u.Email,
				Password: u.Password,
			}
			dbStructure.Users[key] = updatedUser

			err := db.writeDB(dbStructure)
			if err != nil {
				return User{}, err
			}

			return User{
				Id:    updatedUser.Id,
				Email: updatedUser.Email,
			}, nil
		}
	}

	return User{}, errors.New("user not found")
}

func (db *DB) RevokeToken(token string) error {
	dbStructure, err := db.loadDB()
	if err != nil {
		return err
	}

	size := len(dbStructure.Tokens)
	newRevokedToken := Token{
		Id:         token,
		RevokeTime: time.Now().String(),
	}
	dbStructure.Tokens[size+1] = newRevokedToken
	err = db.writeDB(dbStructure)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) IsTokenRevoked(token string) (bool, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return false, err
	}

	for _, value := range dbStructure.Tokens {
		if value.Id == token {
			return true, nil
		}
	}

	return false, nil
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

// GetUsers returns all users in the database
func (db *DB) GetUsers() ([]User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return []User{}, err
	}

	var respSlice []User
	for _, v := range dbStructure.Users {
		respSlice = append(respSlice, v)
	}
	sort.Slice(respSlice, func(i, j int) bool { return respSlice[i].Id < respSlice[j].Id })

	return respSlice, nil
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
	sort.Slice(respSlice, func(i, j int) bool { return respSlice[i].Id < respSlice[j].Id })

	return respSlice, nil
}

func (db *DB) GetUser(v string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	id, err := strconv.Atoi(v)
	if err != nil {
		return User{}, err
	}

	for key, value := range dbStructure.Users {
		if value.Id == id {
			return dbStructure.Users[key], nil
		}
	}

	err = errors.New("user does not exist")
	return User{}, err
}

func (db *DB) GetChirp(v string) (Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	id, err := strconv.Atoi(v)
	if err != nil {
		fmt.Println("Cannot convert to int")
		return Chirp{}, err
	}

	for key, value := range dbStructure.Chirps {
		if value.Id == id {
			return dbStructure.Chirps[key], nil
		}
	}

	err = errors.New("chirp does not exist")
	return Chirp{}, err
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
		Users:  map[int]User{},
		Tokens: map[int]Token{},
	}

	file, err := os.OpenFile(db.path, os.O_RDONLY, 0o755)
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

	file, err := os.OpenFile(db.path, os.O_RDWR, 0o755)
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
