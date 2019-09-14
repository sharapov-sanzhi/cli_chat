package room

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/streadway/amqp"
)

type Config struct {
	DBUserName string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
	RMQHost    string
	RMQPort    string
}

type Message struct {
	UserID    int64
	UserName  string
	Text      string
	CreatedAt string
}

type RoomManager struct {
	db          *sqlx.DB
	ch          *amqp.Channel
	userID      int64
	userName    string
	usersOnline map[string]chan struct{}
	failOnError func(error, string)
	printMsg    func(string, string)
	conf        *Config
	exit        chan struct{}
}

func NewRoomManager(db *sqlx.DB, userName string, failOnError func(error, string), conf *Config, exit chan struct{}) *RoomManager {
	rm := &RoomManager{}

	rm.db = db
	rm.ch = nil
	rm.userName = userName
	rm.usersOnline = make(map[string]chan struct{})
	rm.failOnError = failOnError
	rm.printMsg = printMsg
	rm.conf = conf
	rm.exit = exit

	return rm
}

func (rm *RoomManager) StartRoomManager() {
	err := rm.LoginOrRegister()
	rm.failOnError(err, "Failed to log in or register")
	fmt.Printf("[%s] has been logged in\n", rm.userName)

	conn, err := amqp.Dial("amqp://guest:guest@" + rm.conf.RMQHost + ":" + rm.conf.RMQPort + "/")
	rm.failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	rm.ch, err = conn.Channel()
	rm.failOnError(err, "Failed to open a channel")
	defer rm.ch.Close()

	/* receiving */
	rm.receiver()

	/* sending */
	msg := make(chan string, 1)
	go rm.handleInput(msg)
	go rm.sender(msg)

	//	checking users online/offline every 3 sec
	rm.monitorUsers()
}

func printMsg(msg, at string) {
	fmt.Printf("%s %s", at, msg)
}
