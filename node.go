package main

import (
	"net"
	"time"
)

type node struct {
	id             string
	addr           *net.UDPAddr
	lastQuery      int64
	lastReply      int64
	hasEverReplied bool
}

func (n *node) isGood() bool {
	if n.lastQuery > time.Now().Add(-15*time.Minute).Unix() {
		return true
	}
	if n.hasEverReplied && n.lastReply > time.Now().Add(-15*time.Minute).Unix() {
		return true
	}
	return false
}
