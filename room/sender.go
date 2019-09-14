package room

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"github.com/streadway/amqp"
)

func (rm *RoomManager) handleInput(msg chan string) {
	for {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			s := scanner.Text()
			msg <- s
		}
	}
}

func (rm *RoomManager) sender(msg chan string) {
	for {
		select {
		case <-rm.exit:
			return
		case s := <-msg:
			if s != "" {
				message := &Message{
					UserID:    rm.userID,
					UserName:  rm.userName,
					Text:      s,
					CreatedAt: time.Now().Format("15:04:05"),
				}
				msgJSON, err := json.Marshal(message)
				rm.failOnError(err, "Failed to encode message")
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
}
