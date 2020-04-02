package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
)

/******************** DATA STRUCTS ***********************/

// Player represents a player
type Player struct {
	ID   int
	Name string
}

// Game represents a game object in the database
type Game struct {
	ID       int
	Channels [2]chan string
	Player1  string
	Player2  string
}

// Move represents a move in the game
type Move struct {
	GameID   int    `json:"game_id"`
	Player   int    `json:"player"`
	MoveData string `json:"move"`
	GameOver bool   `json:"game_over"`
}

var (
	gameData      map[int]*Game
	gameDataMutex sync.RWMutex

	idCounter      int
	idCounterMutex sync.RWMutex
)

/************** MAIN FUNCTION **********************/

func main() {
	// Initialise global variables
	gameData = make(map[int]*Game)

	http.HandleFunc("/", hello)

	http.HandleFunc("/move", move)
	http.HandleFunc("/create", createGame)
	http.HandleFunc("/join", joinGame)
	http.HandleFunc("/wait", checkJoinGame)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("could not start server")
	}
}

/************************ HANDLERS ************************/

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Big Tac Toe")
}

func move(w http.ResponseWriter, r *http.Request) {
	jsonData := readJsonFromRequest(r)

	var move Move
	err := json.Unmarshal(jsonData, &move)
	if err != nil {
		log.Println("could not decode json data into Move struct")
	}

	gameDataMutex.RLock()
	game := gameData[move.GameID]
	gameDataMutex.RUnlock()
	sendChan := game.Channels[move.Player]
	recChan := game.Channels[1-move.Player]
	sendChan <- move.MoveData

	if move.GameOver {
		fmt.Fprintf(w, "Game over!")
		return
	}

	response := <-recChan
	fmt.Fprintf(w, response)
}

// Creates a game with one player
func createGame(w http.ResponseWriter, r *http.Request) {
	jsonData := readJsonFromRequest(r)

	var playerData struct {
		Player string `json:"player"`
	}
	err := json.Unmarshal(jsonData, &playerData)
	if err != nil {
		log.Println("could not decode json data into Move struct")
	}

	newGameID := generateID()
	newGame := &Game{
		ID:       newGameID,
		Channels: [2]chan string{make(chan string, 1), make(chan string, 1)},
		Player1:  playerData.Player,
	}
	gameDataMutex.Lock()
	gameData[newGameID] = newGame
	gameDataMutex.Unlock()

	fmt.Fprintf(w, strconv.Itoa(newGame.ID))
}

// Adds a 2nd player to a game, given the game id
func joinGame(w http.ResponseWriter, r *http.Request) {
	jsonData := readJsonFromRequest(r)

	var join struct {
		GameID int    `json:"game_id"`
		Player string `json:"player"`
	}
	err := json.Unmarshal(jsonData, &join)
	if err != nil {
		log.Println("could not decode json data")
	}

	gameDataMutex.RLock()
	game, found := gameData[join.GameID]
	gameDataMutex.RUnlock()
	if !found {
		fmt.Fprintf(w, "error: could not find game ID in checkJoinGame endpoint")
		return
	}

	game.Player2 = join.Player
	game.Channels[0] <- join.Player
	fmt.Fprintf(w, game.Player1)
}

// Returns if the 2nd player has joined the game yet
func checkJoinGame(w http.ResponseWriter, r *http.Request) {
	jsonData := readJsonFromRequest(r)

	var gameID struct {
		GameID int `json:"game_id"`
	}
	err := json.Unmarshal(jsonData, &gameID)
	if err != nil {
		log.Println("could not decode json data")
	}

	// check if player 2 joined
	gameDataMutex.RLock()
	game, found := gameData[gameID.GameID]
	gameDataMutex.RUnlock()
	if !found {
		log.Println("error: could not find game ID in checkJoinGame endpoint")
		return
	}
	p2 := <-game.Channels[0]

	fmt.Fprintf(w, p2) // send player2's name
}

/************** Helper functions *******************/

func generateID() int {
	idCounterMutex.Lock()
	defer idCounterMutex.Unlock()
	idCounter++
	return idCounter
}

func readJsonFromRequest(r *http.Request) []byte {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("could not read json from body")
	}
	return data
}
