package room

import (
	"encoding/json"
	"fmt"
)

func (rm *RoomManager) receiver() {
	err := rm.ch.ExchangeDeclare(
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
				rm.printMsg(fmt.Sprintf("[%s]: %s\n", message.UserName, message.Text), message.CreatedAt)
			}
		}
	}()
}
