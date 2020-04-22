package main

import (
	"crypto/rand"
	"encoding/binary"
	"io/ioutil"
	"net"
	"os"

	"github.com/zeebo/bencode"
)

func (dht *DHT) Load(filename string) {
	f, _ := os.Open(filename)
	defer f.Close()
	data, _ := ioutil.ReadAll(f)
	var obj map[string]interface{}
	bencode.DecodeBytes(data, &obj)

	id, _ := obj["node-id"].(string)
	if len(id) != 20 {
		buf := make([]byte, 20)
		_, err := rand.Read(buf)
		if err != nil {
			return
		}
		id = string(buf)
	}
	dht.id = id
	nodes, _ := obj["nodes"].([]interface{})
	for _, node := range nodes {
		node2, _ := node.(string)
		if len(node2) != 6 {
			continue
		}
		ip := net.IP(node2[:4])
		port := int(binary.BigEndian.Uint16([]byte(node2[4:])))
		addr := &net.UDPAddr{
			IP:   ip,
			Port: port,
		}
		dht.insert(addr)
	}
}

func (dht *DHT) Save(filename string) error {
	var nodes []interface{}
	for _, bucket := range dht.buckets {
		for _, node := range bucket.nodes {
			if node.addr == nil {
				continue
			}
			addr := make([]byte, 6)
			copy(addr[:4], node.addr.IP)
			binary.BigEndian.PutUint16(addr[4:], uint16(node.addr.Port))
			nodes = append(nodes, addr)
		}
	}
	data, err := bencode.EncodeBytes(map[string]interface{}{
		"node-id": dht.id,
		"nodes":   nodes,
	})
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}
