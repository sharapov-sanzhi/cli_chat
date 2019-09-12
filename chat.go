package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"./room"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	conf := room.Config{}
	file, err := os.Open("./conf.json")
	failOnError(err, "Failed to open config file")
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&conf)
	failOnError(err, "Failed to decode config file")

	fmt.Printf("Welcome to the public chat!\n\n")
	fmt.Print("Please, enter your name: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	userName := input.Text()
	if userName == "" || len(userName) > 15 {
		fmt.Println("Incorrect name: empty or very long")
		return
	}

	db, err := sqlx.Connect("mysql", conf.DBUserName+":"+conf.DBPassword+"@tcp("+conf.DBHost+":"+conf.DBPort+")/"+conf.DBName+"?charset=utf8&parseTime=True")
	failOnError(err, "Failed to open DB")
	defer db.Close()

	room := room.NewRoomManager(db, userName, failOnError, &conf)
	go room.StartRoomManager()

	exit := make(chan struct{})

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		<-sigs
		fmt.Println("Exiting...")
		failOnError(room.DeleteUserFromDB(), "Failed to delete user from DB")
		close(exit)
	}()

	<-exit
}
