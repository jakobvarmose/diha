package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/zeebo/bencode"
)

type DHT struct {
	s         *net.UDPConn
	id        string
	callbacks [1024]chan map[string]interface{}
	txid      int
	buckets   []bucket

	mu sync.Mutex
}

type bucket struct {
	nodes [8]*node
}

func NewDHT(addr string) (*DHT, error) {
	id := make([]byte, 20)
	_, err := rand.Read(id)
	if err != nil {
		return nil, err
	}
	udpaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	s, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		return nil, err
	}
	dht := &DHT{
		s:       s,
		id:      string(id),
		buckets: make([]bucket, 32),
	}
	return dht, nil
}

func (dht *DHT) Start() {
	go func() {
		for {
			buf := make([]byte, 512)
			n, source, err := dht.s.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			buf = buf[:n]
			var packet map[string]interface{}
			err = bencode.DecodeBytes(buf, &packet)
			if err != nil {
				continue
			}
			dht.handlePacket(packet, source)
		}
	}()
}

func (dht *DHT) Close() error {
	return dht.s.Close()
}

func (dht *DHT) query(ctx context.Context, d *net.UDPAddr, q string, a map[string]interface{}) (map[string]interface{}, error) {
	dht.mu.Lock()
	dht.txid = (dht.txid + 1) % len(dht.callbacks)
	txid := dht.txid
	if dht.callbacks[txid] != nil {
		dht.callbacks[txid] <- nil
		dht.callbacks[txid] = nil
	}
	ch := make(chan map[string]interface{}, 1)
	dht.callbacks[txid] = ch
	dht.mu.Unlock()

	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(txid))
	a["id"] = dht.id
	obj := map[string]interface{}{
		"y": "q",
		"t": string(buf),
		"q": q,
		"a": a,
	}
	data, err := bencode.EncodeBytes(obj)
	if err != nil {
		return nil, err
	}
	_, err = dht.s.WriteToUDP(data, d)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-ch:
		if reply == nil {
			return nil, context.Canceled
		}
		return reply, nil
	}
}

func (dht *DHT) handleReply(packet map[string]interface{}, source *net.UDPAddr) {
	buf, _ := packet["t"].(string)
	if len(buf) != 2 {
		return
	}
	txid := binary.BigEndian.Uint16([]byte(buf))
	if int(txid) >= len(dht.callbacks) {
		return
	}
	reply, _ := packet["r"].(map[string]interface{})

	dht.mu.Lock()
	defer dht.mu.Unlock()
	if dht.callbacks[txid] == nil {
		return
	}
	dht.callbacks[txid] <- reply
	dht.callbacks[txid] = nil
}

func (dht *DHT) reply(dest *net.UDPAddr, txid interface{}, packet map[string]interface{}) error {
	packet["id"] = dht.id
	obj := map[string]interface{}{
		"y": "r",
		"t": txid,
		"r": packet,
	}
	data, err := bencode.EncodeBytes(obj)
	if err != nil {
		return err
	}
	_, err = dht.s.WriteToUDP(data, dest)
	return err
}

func (dht *DHT) handleQuery(packet map[string]interface{}, source *net.UDPAddr) {
	switch packet["q"] {
	case "ping":
		dht.reply(source, packet["t"], map[string]interface{}{})
	case "find_node":
	case "get_peers":
	case "announce_peer":
	}
}

func (dht *DHT) handlePacket(packet map[string]interface{}, source *net.UDPAddr) {
	switch packet["y"] {
	case "q":
		dht.handleQuery(packet, source)
	case "e":
	// Ignore errors
	case "r":
		dht.handleReply(packet, source)
	}
}

func (dht *DHT) update(id string, addr *net.UDPAddr, query bool) {
	index := bucketIndex(distance(dht.id, id))
	if index > len(dht.buckets)-1 {
		index = len(dht.buckets) - 1
	}
	for _, node := range dht.buckets[index].nodes {
		if node == nil {
			continue
		}
		if node.addr.String() == addr.String() {
			if query {
				node.lastQuery = time.Now().Unix()
			} else {
				node.lastReply = time.Now().Unix()
				node.hasEverReplied = true
			}
			return
		}
	}
	for i, n := range dht.buckets[index].nodes {
		if n == nil || !n.isGood() {
			dht.buckets[index].nodes[i] = &node{
				id:   id,
				addr: addr,
			}
			return
		}
	}
}

func (dht *DHT) insert(addr *net.UDPAddr) {
	fmt.Printf("%v\n", addr)
}

func main() {
	dht, err := NewDHT(":7777")
	if err != nil {
		panic(err)
	}
	defer dht.Close()
	dht.Load("dht.state")
	dht.Start()
	dht.Save("dht2.state")

	addr2, err := net.ResolveUDPAddr("udp", "10.0.0.100:60386")
	if err != nil {
		panic(err)
	}
	data, err := dht.query(context.TODO(), addr2, "ping", map[string]interface{}{
		"id": "abcdefghij0123456789",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", data)
}
