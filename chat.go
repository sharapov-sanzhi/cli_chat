package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"./room"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	conf := room.Config{}
	file, err := os.Open("./conf.json")
	failOnError(err, "Failed to open config file")
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&conf)
	failOnError(err, "Failed to read config file")

	fmt.Printf("Welcome to the public chat!\n\n")
	fmt.Print("Please, enter your name: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	userName := input.Text()
	if userName == "" || len(userName) > 15 {
		fmt.Println("Incorrect name: empty or too long")
		return
	}

	db, err := sqlx.Connect("postgres", room.DBConn(&conf))
	failOnError(err, "Failed to open DB")
	defer db.Close()

	exit := make(chan struct{})

	room := room.NewRoomManager(db, userName, failOnError, &conf, exit)
	go room.StartRoomManager()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	<-sigs
	fmt.Println("Exiting...")
	failOnError(room.Logout(), "Failed to logout")
	close(exit)

	<-exit
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
