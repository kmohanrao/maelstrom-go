package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Topology map[string][]string
type TopologyMessage struct {
	Type  string   `json:"type"`
	Topo  Topology `json:"topology,omitempty"`
	MsgID int      `json:"msg_id"`
}

type NeoNode struct {
	*maelstrom.Node
	messages map[float64]struct{}
	data     []float64
	topology Topology
	mu       sync.RWMutex
}

func NewNeoNode() NeoNode {
	return NeoNode{
		Node:     maelstrom.NewNode(),
		messages: make(map[float64]struct{}),
		data:     []float64{},
	}
}

func main() {
	m := NewNeoNode()

	m.Handle("broadcast", m.handleBroadcast)

	m.Handle("broadcast_ok", m.ignoreMessage)

	m.Handle("read", m.handleRead)

	m.Handle("topology", m.handleTopology)

	if err := m.Run(); err != nil {
		log.Fatal(err)
	}

}

func (n *NeoNode) dataExists(k float64) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.messages[k]
	return ok
}

func (n *NeoNode) handleBroadcast(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	src := msg.Src
	value := body["message"].(float64)
	// _, ok := n.messages[value]

	if !n.dataExists(value) {
		n.mu.Lock()
		n.messages[value] = struct{}{}
		n.mu.Unlock()
		n.data = append(n.data, value)
		for _, neighbour := range n.topology[n.ID()] {
			if strings.Compare(neighbour, src) == 0 {
				continue
			}

			fmt.Fprintln(os.Stderr, neighbour)
			n.Send(neighbour, body)
		}
	}
	body["type"] = "broadcast_ok"

	delete(body, "message")
	return n.Reply(msg, body)
}

func (n *NeoNode) handleRead(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "read_ok"
	body["messages"] = n.data

	return n.Reply(msg, body)
}

func (n *NeoNode) handleTopology(msg maelstrom.Message) error {
	var body TopologyMessage
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// body["type"] = "topology_ok"
	body.Type = "topology_ok"
	// delete(body, "topology")
	n.topology = body.Topo
	body.Topo = nil

	fmt.Fprintln(os.Stderr, n.topology)
	return n.Reply(msg, body)
}

func (n *NeoNode) ignoreMessage(msg maelstrom.Message) error {
	return nil
}
