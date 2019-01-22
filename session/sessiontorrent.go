package session

import (
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/nictuku/dht"
)

type Torrent struct {
	id           uint64
	port         uint16
	dhtAnnouncer *dhtAnnouncer
	session      *Session
	torrent      *torrent
	removed      chan struct{}
}

func (t *Torrent) ID() uint64 {
	return t.id
}

func (t *Torrent) Name() string {
	return t.torrent.Name()
}

func (t *Torrent) InfoHash() string {
	return t.torrent.InfoHash()
}

func (t *Torrent) Stats() Stats {
	return t.torrent.Stats()
}

func (t *Torrent) Trackers() []Tracker {
	return t.torrent.Trackers()
}

func (t *Torrent) Peers() []Peer {
	return t.torrent.Peers()
}

func (t *Torrent) Port() uint16 {
	return t.port
}

func (t *Torrent) Start() error {
	subBucket := strconv.FormatUint(t.id, 10)
	err := t.session.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(torrentsBucket).Bucket([]byte(subBucket))
		return b.Put([]byte("started"), []byte("1"))
	})
	if err != nil {
		return err
	}
	t.torrent.Start()
	if !t.torrent.Stats().Private {
		t.session.mPeerRequests.Lock()
		t.session.dhtPeerRequests[dht.InfoHash(t.torrent.InfoHashBytes())] = struct{}{}
		t.session.mPeerRequests.Unlock()
	}
	return nil
}

func (t *Torrent) Stop() error {
	subBucket := strconv.FormatUint(t.id, 10)
	err := t.session.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(torrentsBucket).Bucket([]byte(subBucket))
		return b.Put([]byte("started"), []byte("0"))
	})
	if err != nil {
		return err
	}
	t.torrent.Stop()
	return nil
}