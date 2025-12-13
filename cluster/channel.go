// Copyright 2018 Prometheus Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"log/slog"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/memberlist"

	"github.com/prometheus/alertmanager/cluster/clusterpb"
)

// Channel allows clients to send messages for a specific state type that will be
// broadcasted in a best-effort manner.
type Channel struct {
	key          string
	send         func([]byte)
	peers        func() []*memberlist.Node
	sendOversize func(*memberlist.Node, []byte) error

	msgc   chan []byte
	logger *slog.Logger
}

// NewChannel creates a new Channel struct, which handles sending normal and
// oversize messages to peers.
func NewChannel(
	key string,
	send func([]byte),
	peers func() []*memberlist.Node,
	sendOversize func(*memberlist.Node, []byte) error,
	logger *slog.Logger,
	stopc <-chan struct{},
) *Channel {
	c := &Channel{
		key:          key,
		send:         send,
		peers:        peers,
		logger:       logger,
		msgc:         make(chan []byte, 200),
		sendOversize: sendOversize,
	}

	go c.handleOverSizedMessages(stopc)

	return c
}

// handleOverSizedMessages prevents memberlist from opening too many parallel
// TCP connections to its peers.
func (c *Channel) handleOverSizedMessages(stopc <-chan struct{}) {
	var wg sync.WaitGroup
	for {
		select {
		case b := <-c.msgc:
			for _, n := range c.peers() {
				wg.Add(1)
				go func(n *memberlist.Node) {
					defer wg.Done()
					if err := c.sendOversize(n, b); err != nil {
						c.logger.Debug("failed to send reliable", "key", c.key, "node", n, "err", err)
						return
					}
				}(n)
			}

			wg.Wait()
		case <-stopc:
			return
		}
	}
}

// Broadcast enqueues a message for broadcasting.
func (c *Channel) Broadcast(b []byte) {
	b, err := proto.Marshal(&clusterpb.Part{Key: c.key, Data: b})
	if err != nil {
		return
	}

	if OversizedMessage(b) {
		select {
		case c.msgc <- b:
		default:
			c.logger.Debug("oversized gossip channel full")
		}
	} else {
		c.send(b)
	}
}

// OversizedMessage indicates whether or not the byte payload should be sent
// via TCP.
func OversizedMessage(b []byte) bool {
	return len(b) > MaxGossipPacketSize/2
}
