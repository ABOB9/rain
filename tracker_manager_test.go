package rain

import "testing"

func TestManager(t *testing.T) {
	const url = "udp://127.0.0.1:6969"

	c := &dummyClient{peerID: PeerID{}}
	m := newManager(c)
	tr, err := m.NewTracker(url)
	if err != nil {
		t.Fatal(err)
	}
	if m.trackers[url].count != 1 {
		t.Fatal("not 1")
	}
	tr2, err := m.NewTracker(url)
	if err != nil {
		t.Fatal(err)
	}
	if m.trackers[url].count != 2 {
		t.Fatal("not 2")
	}
	if tr != tr2 {
		t.Fatal("not equal")
	}
	tr.Close()
	if m.trackers[url].count != 1 {
		t.Fatal("not 1")
	}
	tr2.Close()
	if _, ok := m.trackers[url]; ok {
		t.Fatal("ok")
	}
}

type dummyClient struct {
	peerID PeerID
}

func (c *dummyClient) PeerID() PeerID { return c.peerID }
func (c *dummyClient) Port() uint16   { return 6881 }