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
)

type Topology map[string][]string
type TopologyMessage struct {
	Type  string   `json:"type"`
	Topo  Topology `json:"topology,omitempty"`
	MsgID int      `json:"msg_id"`
}
type NeoNode struct {
	*maelstrom.Node
	// messages           map[float64]struct{}
	data float64
	// topology           Topology
	// mu                 sync.RWMutex
	// lastGossippedIndex int
	// nextIndex          int
	stream chan float64
	kv     *maelstrom.KV
}

func NewNeoNode() NeoNode {
	n := NeoNode{
		Node:   maelstrom.NewNode(),
		stream: make(chan float64, ChannelLength),
	}

	n.kv = maelstrom.NewSeqKV(n.Node)

	return n
}

func main() {

	m := NewNeoNode()

	m.Handle("add", m.handleAdd)

	m.Handle("read", m.handleRead)

	if err := m.Run(); err != nil {
		log.Fatal(err)
	}

}

func (n *NeoNode) Run() error {
	// interval := time.Duration(GossipInterval+rand.Intn(25)) * time.Millisecond
	// ticker := time.NewTicker(interval)
	// defer ticker.Stop()

	log.Print("start the node")
	go n.updateStore()

	if err := n.Node.Run(); err != nil {
		return err
	}
	return nil
}

func (n *NeoNode) handleRead(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	key := "foo"
	body["type"] = "read_ok"
	v, err := n.kv.ReadInt(context.Background(), key)
	if err != nil {
		log.Printf("no value found for %s", key)
		v = 0
	}

	body["value"] = v

	return n.Reply(msg, body)
}

func (n *NeoNode) handleAdd(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	value := body["delta"].(float64)
	// n.data += value

	n.stream <- value
	body["type"] = "add_ok"

	delete(body, "delta")
	return n.Reply(msg, body)
}

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
