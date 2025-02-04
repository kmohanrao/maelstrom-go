package main

import (
	"context"
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	GossipInterval = 40
	ChannelLength  = 10
	MaxMessages    = 10
)

type Topology map[string][]string
type TopologyMessage struct {
	Type  string   `json:"type"`
	Topo  Topology `json:"topology,omitempty"`
	MsgID int      `json:"msg_id"`
}
type NeoNode struct {
	*maelstrom.Node
	stream      chan float64
	dataStore   map[string][]int
	commitStore map[string]int
	kv          *maelstrom.KV
}

func NewNeoNode() NeoNode {
	n := NeoNode{
		Node:        maelstrom.NewNode(),
		stream:      make(chan float64, ChannelLength),
		dataStore:   make(map[string][]int),
		commitStore: make(map[string]int),
	}

	n.kv = maelstrom.NewSeqKV(n.Node)

	return n
}

func main() {

	m := NewNeoNode()

	m.Handle("send", m.handleSend)

	m.Handle("poll", m.handlePoll)

	m.Handle("commit_offsets", m.handleCommits)

	m.Handle("list_committed_offsets", m.handleListCommits)

	if err := m.Run(); err != nil {
		log.Fatal(err)
	}

}

func (n *NeoNode) Run() error {
	log.Print("start the node")
	go n.updateStore()

	if err := n.Node.Run(); err != nil {
		return err
	}
	return nil
}

type pollRequest struct {
	Type    string             `json:"type"`
	Offsets map[string]float64 `json:"offsets"`
}

type pollResponse struct {
	Type     string             `json:"type"`
	Messages map[string][][]int `json:"msgs"`
}

func (n *NeoNode) handlePoll(msg maelstrom.Message) error {
	var request pollRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return err
	}

	var response pollResponse
	response.Type = "poll_ok"
	for key, value := range request.Offsets {
		messages := [][]int{}
		for i := int(value); i < len(n.dataStore[key]) || i-int(value) < MaxMessages; i++ {
			messages = append(messages, []int{i, n.dataStore[key][i]})
		}
		response.Messages[key] = messages
	}

	return n.Reply(msg, response)
}

func (n *NeoNode) handleSend(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "send_ok"
	key := body["key"].(string)
	value := body["msg"].(float64)

	array, ok := n.dataStore[key]
	if !ok {
		array = []int{}
		n.dataStore[key] = array
	}

	array = append(array, int(value))

	body["offset"] = len(array)
	delete(body, "key")
	delete(body, "msg")
	return n.Reply(msg, body)
}

func (n *NeoNode) handleCommits(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// _, ok := n.messages[value]

	body["type"] = "commit_offsets_ok"

	delete(body, "offsets")
	return n.Reply(msg, body)
}

func (n *NeoNode) handleListCommits(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// _, ok := n.messages[value]

	body["type"] = "list_committed_offsets_ok"
	body["offsets"] = make(map[string]int)

	delete(body, "offsets")
	return n.Reply(msg, body)
}

// func (n *NeoNode) getNextOffset(key string) int {
// 	current, err := n.kv.ReadInt(context.Background(), key)
// 	if err != nil {
// 		log.Printf("unable to read key from kv: %s", key)
// 		current = 0
// 	}

// 	for n.kv.CompareAndSwap(context.Background(), key, current, current+1, true) != nil {
// 		current, err = n.kv.ReadInt(context.Background(), key)
// 		if err != nil {
// 			log.Printf("unable to read key from kv: %s", key)
// 			current = 0
// 		}
// 	}
// 	return current + 1
// }

func (n *NeoNode) updateStore() {
	key := "foo"
	for v := range n.stream {
		current, err := n.kv.ReadInt(context.Background(), key)
		if err != nil {
			log.Printf("unable to read key from kv: %s", key)
			current = 0
		}

		for n.kv.CompareAndSwap(context.Background(), key, current, current+int(v), true) != nil {
			current, err = n.kv.ReadInt(context.Background(), key)
			if err != nil {
				log.Printf("unable to read key from kv: %s", key)
				current = 0
			}
		}
	}
}
