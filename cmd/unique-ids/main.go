package main

import (
	"encoding/json"
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	m := maelstrom.NewNode()
	counter := 1

	m.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "generate_ok"
		body["id"] = fmt.Sprintf("%s%d", m.ID(), counter)
		counter += 1

		return m.Reply(msg, body)
	})
	if err := m.Run(); err != nil {
		log.Fatal(err)
	}

}
