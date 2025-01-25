package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	GossipInterval = 40
	ChannelLength  = 100
)

type Topology map[string][]string
type TopologyMessage struct {
	Type  string   `json:"type"`
	Topo  Topology `json:"topology,omitempty"`
	MsgID int      `json:"msg_id"`
}

type NeoNode struct {
	*maelstrom.Node
	messages           map[float64]struct{}
	data               []float64
	topology           Topology
	mu                 sync.RWMutex
	lastGossippedIndex int
	nextIndex          int
	stream             chan float64
}

func NewNeoNode() NeoNode {
	return NeoNode{
		Node:     maelstrom.NewNode(),
		messages: make(map[float64]struct{}),
		data:     []float64{},
		stream:   make(chan float64, ChannelLength),
	}
}

func main() {
	m := NewNeoNode()

	m.Handle("broadcast", m.handleBroadcast)

	m.Handle("gossip", m.handleGossip)

	m.Handle("read", m.handleRead)

	m.Handle("topology", m.handleTopology)

	if err := m.Run(); err != nil {
		log.Fatal(err)
	}

}

func (n *NeoNode) Run() error {
	interval := time.Duration(GossipInterval+rand.Intn(25)) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			n.gossip()
		}
	}()

	go n.updateStore()

	if err := n.Node.Run(); err != nil {
		return err
	}
	return nil
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

	value := body["message"].(float64)
	// _, ok := n.messages[value]

	n.stream <- value
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
	// n.topology = body.Topo

	jsonData := `{
		"n0": ["n1", "n2", "n3", "n4", "n5", "n6"],
		"n1": ["n0", "n7", "n8", "n9", "n10", "n11", "n12"],
		"n2": ["n0", "n13", "n14", "n15", "n16", "n17", "n18"],
		"n3": ["n0", "n19", "n20", "n21", "n22", "n23", "n24"],
		"n4": ["n0"],
		"n5": ["n0"],
		"n6": ["n0"],
		"n7": ["n1"],
		"n8": ["n1"],
		"n9": ["n1"],
		"n10": ["n1"],
		"n11": ["n1"],
		"n12": ["n1"],
		"n13": ["n2"],
		"n14": ["n2"],
		"n15": ["n2"],
		"n16": ["n2"],
		"n17": ["n2"],
		"n18": ["n2"],
		"n19": ["n3"],
		"n20": ["n3"],
		"n21": ["n3"],
		"n22": ["n3"],
		"n23": ["n3"],
		"n24": ["n3"]
	}`

	// Declare a variable of type Topology
	var topology Topology

	// Unmarshal JSON into the Topology variable
	err := json.Unmarshal([]byte(jsonData), &topology)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}
	n.topology = topology
	body.Topo = nil

	fmt.Fprintln(os.Stderr, n.topology)
	return n.Reply(msg, body)
}

func (n *NeoNode) handleGossip(msg maelstrom.Message) error {
	var body GossipMessageBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	data := body.Data
	// _, ok := n.messages[value]

	for _, value := range data {
		n.stream <- value
	}

	body.Type = "gossip_ok"
	body.Data = nil

	return n.Reply(msg, body)
}

type GossipMessageBody struct {
	maelstrom.MessageBody
	Data []float64 `json:"data,omitempty"`
}

func (n *NeoNode) gossip() error {
	if n.lastGossippedIndex == len(n.data) {
		return nil
	}
	n.nextIndex = len(n.data)
	msgBody := GossipMessageBody{
		MessageBody: maelstrom.MessageBody{
			Type:  "gossip",
			MsgID: n.lastGossippedIndex,
		},
		Data: n.data[n.lastGossippedIndex:n.nextIndex],
	}

	log.Printf("total data: %d, gossipped data: %d\n", n.nextIndex, len(msgBody.Data))
	callbackhandler := n.gossipCallbackHandler(len(n.topology[n.ID()]), 0)
	for _, neighbour := range n.topology[n.ID()] {
		fmt.Fprintln(os.Stderr, neighbour)

		n.RPC(neighbour, msgBody, callbackhandler)
	}

	// n.lastGossippedIndex = newLength
	return nil
}

func (n *NeoNode) gossipCallbackHandler(tc, margin int) maelstrom.HandlerFunc {
	totalCount := tc
	return func(msg maelstrom.Message) error {
		totalCount--
		log.Printf("%s, %d\n", msg.Type(), totalCount)
		if totalCount <= margin {
			n.lastGossippedIndex = n.nextIndex
			log.Printf("new gossip index : %d\n", n.lastGossippedIndex)
		}
		return nil
	}
}

func (n *NeoNode) updateStore() {
	for v := range n.stream {
		if !n.dataExists(v) {
			n.mu.Lock()
			n.messages[v] = struct{}{}
			n.mu.Unlock()
			n.data = append(n.data, v)
		}
	}
}
