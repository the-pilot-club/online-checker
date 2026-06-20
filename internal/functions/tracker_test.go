package functions

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/the-pilot-club/tpcgo"
)

// fakeStore is an in-memory SessionStore for tests.
type fakeStore[T any] struct {
	data map[string]T
}

func newFakeStore[T any]() *fakeStore[T] { return &fakeStore[T]{data: map[string]T{}} }

func (f *fakeStore[T]) Get(_ context.Context, cid string) (T, bool, error) {
	v, ok := f.data[cid]
	return v, ok, nil
}
func (f *fakeStore[T]) Set(_ context.Context, cid string, v T) error { f.data[cid] = v; return nil }
func (f *fakeStore[T]) Delete(_ context.Context, cid string) error   { delete(f.data, cid); return nil }
func (f *fakeStore[T]) Close() error                                 { return nil }

// fakeWebhook records the calls made against it.
type fakeWebhook struct {
	sends  int
	edits  int
	nextID string
}

func (w *fakeWebhook) Send(_ string, _ *discordgo.MessageEmbed) (string, error) {
	w.sends++
	if w.nextID == "" {
		return "msg", nil
	}
	return w.nextID, nil
}
func (w *fakeWebhook) Edit(string, *discordgo.MessageEmbed) error { w.edits++; return nil }

// fakeSource serves a fixed set of CIDs and a fixed data feed.
type fakeSource struct {
	cids []*tpcgo.FCPCIDOnly
	feed *tpcgo.DataFeed
}

func (s fakeSource) GetAllFCPUsersCID() ([]*tpcgo.FCPCIDOnly, error) { return s.cids, nil }
func (s fakeSource) GetVatsimDataFeed() (*tpcgo.DataFeed, error)     { return s.feed, nil }

func cids(ids ...int) []*tpcgo.FCPCIDOnly {
	out := make([]*tpcgo.FCPCIDOnly, len(ids))
	for i, id := range ids {
		out[i] = &tpcgo.FCPCIDOnly{VATSIMCid: id}
	}
	return out
}

func TestPilotTracker_AnnouncesTPCFlightAndStores(t *testing.T) {
	st := newFakeStore[PilotSession]()
	wh := &fakeWebhook{nextID: "m1"}
	src := fakeSource{
		cids: cids(100),
		feed: &tpcgo.DataFeed{Pilots: []tpcgo.Pilot{{CID: 100, Callsign: "TPC123"}}},
	}

	NewTracker[tpcgo.Pilot, PilotSession](st, wh, pilotHandler{}).Run(context.Background(), src)

	if wh.sends != 1 {
		t.Fatalf("expected 1 send, got %d", wh.sends)
	}
	got, ok, _ := st.Get(context.Background(), "100")
	if !ok {
		t.Fatal("expected session stored for cid 100")
	}
	if got.MessageId != "m1" {
		t.Errorf("expected message id m1, got %q", got.MessageId)
	}
}

func TestPilotTracker_IgnoresNonTPCCallsign(t *testing.T) {
	st := newFakeStore[PilotSession]()
	wh := &fakeWebhook{}
	src := fakeSource{
		cids: cids(100),
		feed: &tpcgo.DataFeed{Pilots: []tpcgo.Pilot{{CID: 100, Callsign: "BAW42"}}},
	}

	NewTracker[tpcgo.Pilot, PilotSession](st, wh, pilotHandler{}).Run(context.Background(), src)

	if wh.sends != 0 {
		t.Errorf("expected no announcement, got %d sends", wh.sends)
	}
	if _, ok, _ := st.Get(context.Background(), "100"); ok {
		t.Error("expected no session stored")
	}
}

func TestPilotTracker_EndsSessionWhenOffline(t *testing.T) {
	st := newFakeStore[PilotSession]()
	st.data["100"] = PilotSession{CID: "100", Callsign: "TPC123", MessageId: "m1"}
	wh := &fakeWebhook{}
	src := fakeSource{cids: cids(100), feed: &tpcgo.DataFeed{}} // no pilots online

	NewTracker[tpcgo.Pilot, PilotSession](st, wh, pilotHandler{}).Run(context.Background(), src)

	if wh.edits != 1 {
		t.Errorf("expected 1 edit (end embed), got %d", wh.edits)
	}
	if _, ok, _ := st.Get(context.Background(), "100"); ok {
		t.Error("expected ended session to be deleted")
	}
}

func TestPilotTracker_UpdatesOnNewerRevision(t *testing.T) {
	st := newFakeStore[PilotSession]()
	st.data["100"] = PilotSession{CID: "100", Callsign: "TPC123", MessageId: "m1", RevisionId: "1"}
	wh := &fakeWebhook{}
	src := fakeSource{
		cids: cids(100),
		feed: &tpcgo.DataFeed{Pilots: []tpcgo.Pilot{{
			CID:        100,
			Callsign:   "TPC123",
			FlightPlan: &tpcgo.FlightPlan{RevisionID: 2, Departure: "EGLL"},
		}}},
	}

	NewTracker[tpcgo.Pilot, PilotSession](st, wh, pilotHandler{}).Run(context.Background(), src)

	if wh.edits != 1 {
		t.Errorf("expected 1 edit (update), got %d", wh.edits)
	}
	got, _, _ := st.Get(context.Background(), "100")
	if got.RevisionId != "2" || got.Departure != "EGLL" {
		t.Errorf("expected session updated to revision 2/EGLL, got %+v", got)
	}
}

func TestPilotTracker_NoUpdateForSameRevision(t *testing.T) {
	st := newFakeStore[PilotSession]()
	st.data["100"] = PilotSession{CID: "100", Callsign: "TPC123", MessageId: "m1", RevisionId: "2"}
	wh := &fakeWebhook{}
	src := fakeSource{
		cids: cids(100),
		feed: &tpcgo.DataFeed{Pilots: []tpcgo.Pilot{{
			CID:        100,
			Callsign:   "TPC123",
			FlightPlan: &tpcgo.FlightPlan{RevisionID: 2},
		}}},
	}

	NewTracker[tpcgo.Pilot, PilotSession](st, wh, pilotHandler{}).Run(context.Background(), src)

	if wh.edits != 0 || wh.sends != 0 {
		t.Errorf("expected no webhook calls, got %d sends %d edits", wh.sends, wh.edits)
	}
}

func TestATCTracker_AnnouncesAnyController(t *testing.T) {
	st := newFakeStore[ATCSession]()
	wh := &fakeWebhook{nextID: "a1"}
	src := fakeSource{
		cids: cids(200),
		feed: &tpcgo.DataFeed{Controllers: []tpcgo.Controller{{CID: 200, Callsign: "EGLL_TWR", Frequency: "118.500"}}},
	}

	NewTracker[tpcgo.Controller, ATCSession](st, wh, atcHandler{}).Run(context.Background(), src)

	got, ok, _ := st.Get(context.Background(), "200")
	if !ok || got.MessageId != "a1" {
		t.Fatalf("expected stored ATC session with message id a1, got %+v ok=%v", got, ok)
	}
}

func TestATCTracker_NoUpdateWhileStillOnline(t *testing.T) {
	st := newFakeStore[ATCSession]()
	st.data["200"] = ATCSession{CID: "200", Callsign: "EGLL_TWR", MessageId: "a1"}
	wh := &fakeWebhook{}
	src := fakeSource{
		cids: cids(200),
		feed: &tpcgo.DataFeed{Controllers: []tpcgo.Controller{{CID: 200, Callsign: "EGLL_TWR"}}},
	}

	NewTracker[tpcgo.Controller, ATCSession](st, wh, atcHandler{}).Run(context.Background(), src)

	if wh.sends != 0 || wh.edits != 0 {
		t.Errorf("expected no webhook calls for already-online ATC, got %d sends %d edits", wh.sends, wh.edits)
	}
}

func TestDiscordTime(t *testing.T) {
	got := discordTime(time.Unix(1700000000, 0))
	want := "<t:1700000000:f>"
	if got != want {
		t.Errorf("discordTime = %q, want %q", got, want)
	}
}
