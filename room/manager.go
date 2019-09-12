package room

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/streadway/amqp"
)

func printMsg(msg, at string) {
	fmt.Printf("%s %s", at, msg)
}

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
}

func NewRoomManager(db *sqlx.DB, userName string, failOnError func(error, string), conf *Config) *RoomManager {
	rm := &RoomManager{}

	rm.db = db
	rm.ch = nil
	rm.userName = userName
	rm.usersOnline = make(map[string]chan struct{})
	rm.failOnError = failOnError
	rm.printMsg = printMsg
	rm.conf = conf

	return rm
}

func (rm *RoomManager) InsertUserIntoDB() (id int64, err error) {
	var res sql.Result
	if rm.checkIsEmpty() {
		res, err = rm.db.Exec(`INSERT INTO room (id, name) VALUES (1, ?)`, rm.userName)
	} else {
		res, err = rm.db.Exec(`
		INSERT INTO room (id, name)
		SELECT id + 1, ? AS name
		FROM room
		ORDER BY id DESC
		LIMIT 1
		`, rm.userName)
	}
	if err != nil {
		return id, err
	}
	id, err = res.LastInsertId()
	if err != nil {
		return id, err
	}
	return id, nil
}

func (rm *RoomManager) checkIsEmpty() (ok bool) {
	_ = rm.db.Get(&ok, `SELECT IF (COUNT(*) = 0, TRUE, FALSE) FROM room`)
	return ok
}

func (rm *RoomManager) DeleteUserFromDB() (err error) {
	_, err = rm.db.Exec(`DELETE FROM room WHERE name = ?`, rm.userName)
	return err
}

func (rm *RoomManager) StartRoomManager() {
	id, err := rm.InsertUserIntoDB()
	rm.failOnError(err, "Failed to insert user into DB")
	rm.userID = id
	fmt.Printf("[%s] has been logged in\n", rm.userName)

	conn, err := amqp.Dial("amqp://guest:guest@" + rm.conf.RMQHost + ":" + rm.conf.RMQPort + "/")
	rm.failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	rm.ch, err = conn.Channel()
	rm.failOnError(err, "Failed to open a channel")
	defer rm.ch.Close()

	/* receiving */
	err = rm.ch.ExchangeDeclare(
		"room",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	rm.failOnError(err, "Failed to declare an exchange")

	q, err := rm.ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	rm.failOnError(err, "Failed to declare a queue")

	err = rm.ch.QueueBind(
		q.Name,
		"",
		"room",
		false,
		nil,
	)
	rm.failOnError(err, "Failed to bind a queue")

	msgs, err := rm.ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	rm.failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			message := &Message{}
			err = json.Unmarshal(d.Body, message)
			if rm.userID != message.UserID {
				rm.printMsg(fmt.Sprintf("[%s] sent message: %s\n", message.UserName, message.Text), message.CreatedAt)
			}
		}
	}()

	/* sending */
	msg := make(chan string, 1)

	go func() {
		for {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				s := scanner.Text()
				msg <- s
			}
		}
	}()

	go func() {
		for {
			select {
			case s := <-msg:
				if s != "" {
					message := &Message{
						UserID:    rm.userID,
						UserName:  rm.userName,
						Text:      s,
						CreatedAt: time.Now().Format("15:04:05"),
					}
					msgJSON, err := json.Marshal(message)
					rm.failOnError(err, "")
					err = rm.ch.Publish(
						"room",
						"",
						false,
						false,
						amqp.Publishing{
							ContentType: "text/plain",
							Body:        []byte(msgJSON),
						},
					)
					rm.failOnError(err, "Failed to publish a message")
				}
			default:
				//	nothing to do
			}
		}
	}()

	//	checking users online/offline every 3 sec
	var already = false
	for {
		newUsers := rm.getNewUsersListOnline()
		rm.constructUsers(newUsers)

		exitedUsers := rm.getUsersListExited()
		rm.destructUsers(exitedUsers)

		if len(rm.usersOnline) > 0 {
			already = false
		} else {
			if !already {
				fmt.Println("Now room is empty")
				already = true
			}
		}

		time.Sleep(3 * time.Second)
	}
}

func (rm *RoomManager) constructUsers(list []string) {
	for _, u := range list {
		done := rm.constructUser(u)
		rm.usersOnline[u] = done
	}
}

func (rm *RoomManager) constructUser(name string) chan struct{} {
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			i++
			select {
			case <-done:
				rm.printMsg(fmt.Sprintf("[%s] exited\n", name), time.Now().Format("15:04:05"))
				return
			default:
				if i == 1 {
					rm.printMsg(fmt.Sprintf("[%s] is online\n", name), time.Now().Format("15:04:05"))
				}
			}

			time.Sleep(3 * time.Second)
		}
	}()

	return done
}

func (rm *RoomManager) destructUsers(list []string) {
	for _, u := range list {
		rm.destructUser(u)
	}
}

func (rm *RoomManager) destructUser(u string) {
	close(rm.usersOnline[u])
	delete(rm.usersOnline, u)
}

func (rm *RoomManager) getUsersListOnline() (usersList []string, err error) {
	err = rm.db.Select(&usersList, `SELECT name FROM room WHERE name <> ?`, rm.userName)
	return usersList, err
}

func (rm *RoomManager) getNewUsersListOnline() (newl []string) {
	ul, err := rm.getUsersListOnline()
	rm.failOnError(err, "Failed to get list of users online")
	var wl []string
	for wu := range rm.usersOnline {
		wl = append(wl, wu)
	}
	for _, u := range ul {
		if !contains(wl, u) {
			newl = append(newl, u)
		}
	}

	return newl
}

func (rm *RoomManager) getUsersListExited() (old []string) {
	ul, err := rm.getUsersListOnline()
	rm.failOnError(err, "Failed to get list of users online")
	for wu := range rm.usersOnline {
		if !contains(ul, wu) {
			old = append(old, wu)
		}
	}

	return old
}

func contains(list []string, s string) bool {
	for _, ss := range list {
		if ss == s {
			return true
		}
	}
	return false
}
