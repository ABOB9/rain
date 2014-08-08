package rain

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cenkalti/rain/internal/bitfield"
	"github.com/cenkalti/rain/internal/logger"
	"github.com/cenkalti/rain/internal/protocol"
	"github.com/cenkalti/rain/internal/torrent"
	"github.com/cenkalti/rain/internal/tracker"
)

// transfer represents an active transfer in the program.
type transfer struct {
	rain     *Rain
	tracker  tracker.Tracker
	torrent  *torrent.Torrent
	pieces   []*piece
	bitField bitfield.BitField // pieces that we have
	Finished chan struct{}
	haveC    chan peerHave
	haveCond sync.Cond
	log      logger.Logger
}

func (r *Rain) newTransfer(tor *torrent.Torrent, where string) (*transfer, error) {
	tracker, err := tracker.New(tor.Announce, r.peerID, r.port())
	if err != nil {
		return nil, err
	}
	files, err := allocate(&tor.Info, where)
	if err != nil {
		return nil, err
	}
	pieces := newPieces(&tor.Info, files)
	name := tor.Info.Name
	if len(name) > 8 {
		name = name[:8]
	}
	return &transfer{
		rain:     r,
		tracker:  tracker,
		torrent:  tor,
		pieces:   pieces,
		bitField: bitfield.New(nil, uint32(len(pieces))),
		Finished: make(chan struct{}),
		haveC:    make(chan peerHave),
		haveCond: sync.Cond{L: new(sync.Mutex)},
		log:      logger.New("download " + name),
	}, nil
}

func (t *transfer) InfoHash() protocol.InfoHash { return t.torrent.Info.Hash }
func (t *transfer) Downloaded() int64           { return int64(t.bitField.Count() * t.torrent.Info.PieceLength) }
func (t *transfer) Uploaded() int64             { return 0 } // TODO
func (t *transfer) Left() int64                 { return t.torrent.Info.TotalLength - t.Downloaded() }

func (t *transfer) Run() {
	t.rain.transfersM.Lock()
	t.rain.transfers[t.torrent.Info.Hash] = t
	t.rain.transfersM.Unlock()

	defer func() {
		t.rain.transfersM.Lock()
		delete(t.rain.transfers, t.torrent.Info.Hash)
		t.rain.transfersM.Unlock()
	}()

	peers := make(chan tracker.Peer, tracker.NumWant)
	go t.connecter(peers)

	announceC := make(chan []tracker.Peer)
	go t.tracker.Announce(t, nil, nil, announceC)

	go t.downloader()

	for {
		select {
		case peerAddrs := <-announceC:
			for _, pa := range peerAddrs {
				t.log.Debug("Peer:", pa.TCPAddr())
				select {
				case peers <- pa:
				default:
					<-peers
					peers <- pa
				}
			}
		// case peerConnected TODO
		// case peerDisconnected TODO
		case peerHave := <-t.haveC:
			piece := peerHave.piece
			piece.peersM.Lock()
			piece.peers = append(piece.peers, peerHave.peer)
			piece.peersM.Unlock()

			t.haveCond.L.Lock()
			t.haveCond.Broadcast()
			t.haveCond.L.Unlock()
		}
	}
}

func (t *transfer) connecter(peers chan tracker.Peer) {
	limit := make(chan struct{}, maxPeerPerTorrent)
	for p := range peers {
		limit <- struct{}{}
		go func(peer tracker.Peer) {
			defer func() {
				if err := recover(); err != nil {
					t.log.Critical(err)
				}
				<-limit
			}()
			t.connectToPeer(peer.TCPAddr())
		}(p)
	}
}

func (t *transfer) connectToPeer(addr *net.TCPAddr) {
	t.log.Debugln("Connecting to peer", addr)

	conn, err := net.DialTCP("tcp4", nil, addr)
	if err != nil {
		t.log.Error(err)
		return
	}
	defer conn.Close()
	t.log.Debugf("tcp connection opened to %s", conn.RemoteAddr())

	p := newPeer(conn)

	// Give a minute for completing handshake.
	err = conn.SetDeadline(time.Now().Add(time.Minute))
	if err != nil {
		return
	}

	err = p.sendHandShake(t.torrent.Info.Hash, t.rain.peerID)
	if err != nil {
		p.log.Error(err)
		return
	}

	ih, err := p.readHandShake1()
	if err != nil {
		p.log.Error(err)
		return
	}
	if *ih != t.torrent.Info.Hash {
		p.log.Error("unexpected info_hash")
		return
	}

	id, err := p.readHandShake2()
	if err != nil {
		p.log.Error(err)
		return
	}
	if *id == t.rain.peerID {
		p.log.Debug("Rejected own connection: client")
		return
	}

	p.log.Info("Connected to peer")
	p.Serve(t)
}

func allocate(info *torrent.Info, where string) ([]*os.File, error) {
	if !info.MultiFile() {
		f, err := createTruncateSync(filepath.Join(where, info.Name), info.Length)
		if err != nil {
			return nil, err
		}
		return []*os.File{f}, nil
	}

	// Multiple files
	files := make([]*os.File, len(info.Files))
	for i, f := range info.Files {
		parts := append([]string{where, info.Name}, f.Path...)
		path := filepath.Join(parts...)
		err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0755)
		if err != nil {
			return nil, err
		}
		files[i], err = createTruncateSync(path, f.Length)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func createTruncateSync(path string, length int64) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	err = f.Truncate(length)
	if err != nil {
		return nil, err
	}

	err = f.Sync()
	if err != nil {
		return nil, err
	}

	return f, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}