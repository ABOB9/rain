package torrent

import (
// "time"
)

// TODO implement
func (t *Torrent) announcer() {
	// var nextAnnounce time.Duration
	// var retry = *defaultRetryBackoff

	// announce := func(e Event) {
	// 	r, err := t.Announce(transfer, e, cancel)
	// 	if err != nil {
	// 		r = &AnnounceResponse{Error: err}
	// 		nextAnnounce = retry.NextBackOff()
	// 	} else {
	// 		retry.Reset()
	// 		nextAnnounce = r.Interval
	// 	}
	// 	select {
	// 	case responseC <- r:
	// 	case <-cancel:
	// 		return
	// 	}
	// }

	// announce(StartEvent)
	// for {
	// 	select {
	// 	case <-time.After(nextAnnounce):
	// 		announce(EventNone)
	// 	case e := <-eventC:
	// 		announce(e)
	// 	case <-cancel:
	// 		return
	// 	}
	// }
}

// func (t *Torrent) announcer() {
// 	var startEvent tracker.Event
// 	if t.bitfield.All() {
// 		startEvent = tracker.EventCompleted
// 	} else {
// 		startEvent = tracker.EventStarted
// 	}
// 	tracker.AnnouncePeriodically(t.tracker, t, t.stopC, startEvent, nil, t.announceC)
// }