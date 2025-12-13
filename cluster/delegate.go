// Copyright The Prometheus Authors
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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/memberlist"

	"github.com/prometheus/alertmanager/cluster/clusterpb"
)

const (
	// Maximum number of messages to be held in the queue.
	maxQueueSize = 4096
	fullState    = "full_state"
	update       = "update"
)

// delegate implements memberlist.Delegate and memberlist.EventDelegate
// and broadcasts its peer's state in the cluster.
type delegate struct {
	*Peer

	logger *slog.Logger
	bcast  *memberlist.TransmitLimitedQueue
}

func newDelegate(l *slog.Logger, p *Peer, retransmit int) *delegate {
	bcast := &memberlist.TransmitLimitedQueue{
		NumNodes:       p.ClusterSize,
		RetransmitMult: retransmit,
	}

	d := &delegate{
		logger: l,
		Peer:   p,
		bcast:  bcast,
	}

	go d.handleQueueDepth()

	return d
}

// NodeMeta retrieves meta-data about the current node when broadcasting an alive message.
func (d *delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

// NotifyMsg is the callback invoked when a user-level gossip message is received.
func (d *delegate) NotifyMsg(b []byte) {
	var p clusterpb.Part
	if err := proto.Unmarshal(b, &p); err != nil {
		d.logger.Warn("decode broadcast", "err", err)
		return
	}

	d.mtx.RLock()
	s, ok := d.states[p.Key]
	d.mtx.RUnlock()

	if !ok {
		return
	}
	if err := s.Merge(p.Data); err != nil {
		d.logger.Warn("merge broadcast", "err", err, "key", p.Key)
		return
	}
}

// NotifyConflict is the callback when memberlist encounters two nodes with the same ID.
func (d *delegate) NotifyConflict(existing, other *memberlist.Node) {
	d.logger.Warn("Found conflicting peer IDs", "peer", existing.Name)
}

func (d *delegate) NotifyPingComplete(_ *memberlist.Node, _ time.Duration, _ []byte) {
	d.logger.Debug("Completed ping")
}

func (d *delegate) NotifyAlive(_ *memberlist.Node) error {
	d.logger.Debug("Alive")
	return nil
}

// GetBroadcasts is called when user data messages can be broadcasted.
func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.bcast.GetBroadcasts(overhead, limit)
}

// LocalState is called when gossip fetches local state.
func (d *delegate) LocalState(_ bool) []byte {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	all := &clusterpb.FullState{
		Parts: make([]clusterpb.Part, 0, len(d.states)),
	}

	for key, s := range d.states {
		b, err := s.MarshalBinary()
		if err != nil {
			d.logger.Warn("encode local state", "err", err, "key", key)
			return nil
		}
		all.Parts = append(all.Parts, clusterpb.Part{Key: key, Data: b})
	}
	b, err := proto.Marshal(all)
	if err != nil {
		d.logger.Warn("encode local state", "err", err)
		return nil
	}
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, _ bool) {
	var fs clusterpb.FullState
	if err := proto.Unmarshal(buf, &fs); err != nil {
		d.logger.Warn("merge remote state", "err", err)
		return
	}
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	for _, p := range fs.Parts {
		s, ok := d.states[p.Key]
		if !ok {
			d.logger.Warn("unknown state key", "len", len(buf), "key", p.Key)
			continue
		}
		if err := s.Merge(p.Data); err != nil {
			d.logger.Warn("merge remote state", "err", err, "key", p.Key)
			return
		}
	}
}

// NotifyJoin is called if a peer joins the cluster.
func (d *delegate) NotifyJoin(n *memberlist.Node) {
	d.logger.Debug("NotifyJoin", "node", n.Name, "addr", n.Address())
	d.peerJoin(n)
}

// NotifyLeave is called if a peer leaves the cluster.
func (d *delegate) NotifyLeave(n *memberlist.Node) {
	d.logger.Debug("NotifyLeave", "node", n.Name, "addr", n.Address())
	d.peerLeave(n)
}

// NotifyUpdate is called if a cluster peer gets updated.
func (d *delegate) NotifyUpdate(n *memberlist.Node) {
	d.logger.Debug("NotifyUpdate", "node", n.Name, "addr", n.Address())
	d.peerUpdate(n)
}

// AckPayload implements the memberlist.PingDelegate interface.
func (d *delegate) AckPayload() []byte {
	return []byte{}
}

// handleQueueDepth ensures that the queue doesn't grow unbounded by pruning
// older messages at regular interval.
func (d *delegate) handleQueueDepth() {
	for {
		select {
		case <-d.stopc:
			return
		case <-time.After(15 * time.Minute):
			n := d.bcast.NumQueued()
			if n > maxQueueSize {
				d.logger.Warn("dropping messages because too many are queued", "current", n, "limit", maxQueueSize)
				d.bcast.Prune(maxQueueSize)
			}
		}
	}
}
