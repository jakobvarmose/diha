package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"net"
	"time"
)

var tokenSeed [16]byte

func init() {
	_, err := rand.Read(tokenSeed[:])
	if err != nil {
		panic(err)
	}
}

func getTokenForIndex(addr *net.UDPAddr, index int64) string {
	h := sha1.New()
	h.Write([]byte(addr.String()))
	binary.Write(h, binary.BigEndian, index)
	h.Write(tokenSeed[:])
	return string(h.Sum(nil))
}

func getToken(addr *net.UDPAddr) string {
	index := time.Now().Unix() / 300
	return getTokenForIndex(addr, index)
}

func checkToken(addr *net.UDPAddr, token string) bool {
	index := time.Now().Unix() / 300
	if getTokenForIndex(addr, index) == token {
		return true
	}
	if getTokenForIndex(addr, index-1) == token {
		return true
	}
	return false
}
