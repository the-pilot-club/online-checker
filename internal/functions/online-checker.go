package functions

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/the-pilot-club/online-checker/internal/store"
	"github.com/the-pilot-club/tpcgo"

	_ "github.com/joho/godotenv/autoload"
)

// PilotSession is the persisted state for an online pilot flight. The redis
// tags are used by the Redis store backend and ignored by others.
type PilotSession struct {
	CID           string `redis:"cid"`
	Callsign      string `redis:"callsign"`
	Start         string `redis:"start"`
	MessageId     string `redis:"message_id"`
	AircraftShort string `redis:"aircraft_short"`
	Departure     string `redis:"departure"`
	Arrival       string `redis:"arrival"`
	Route         string `redis:"route"`
	Remarks       string `redis:"remarks"`
	RevisionId    string `redis:"revision_id"`
}

// applyFlightPlan copies the filed flight plan into the session.
func (s *PilotSession) applyFlightPlan(fp *tpcgo.FlightPlan) {
	s.AircraftShort = fp.AircraftShort
	s.Departure = fp.Departure
	s.Arrival = fp.Arrival
	s.Route = fp.Route
	s.Remarks = fp.Remarks
	s.RevisionId = strconv.Itoa(fp.RevisionID)
}

// flightPlanField renders a filed flight plan as an embed field.
func flightPlanField(name string, fp *tpcgo.FlightPlan) *discordgo.MessageEmbedField {
	return field(name, fmt.Sprintf("> A/C: %v\n> DEP: %v\n> ARR: %v\n> Route: %v\n> Remarks: %v",
		fp.AircraftShort, fp.Departure, fp.Arrival, fp.Route, fp.Remarks))
}

// pilotHandler implements Handler for VATSIM pilots flying TPC callsigns.
type pilotHandler struct{}

func (pilotHandler) Username() string { return "TPC Flight Tracking" }

// ShouldAnnounce limits announcements to TPC callsigns.
func (pilotHandler) ShouldAnnounce(p tpcgo.Pilot) bool {
	return strings.Contains(p.Callsign, "TPC")
}

func (pilotHandler) Online(feed *tpcgo.DataFeed) map[string]tpcgo.Pilot {
	online := make(map[string]tpcgo.Pilot, len(feed.Pilots))
	for _, p := range feed.Pilots {
		online[strconv.Itoa(p.CID)] = p
	}
	return online
}

func (pilotHandler) StartEmbed(p tpcgo.Pilot, now time.Time) (*discordgo.MessageEmbed, PilotSession) {
	embed := newEmbed("A flight has started!",
		field("Callsign", p.Callsign),
		field("Start Time", discordTime(now)),
	)
	session := PilotSession{
		CID:      strconv.Itoa(p.CID),
		Callsign: p.Callsign,
		Start:    discordTime(now),
	}
	if p.FlightPlan != nil {
		embed.Fields = append(embed.Fields, flightPlanField("Filed Flight Plan", p.FlightPlan))
		session.applyFlightPlan(p.FlightPlan)
	}
	return embed, session
}

func (pilotHandler) EndEmbed(s PilotSession, now time.Time) *discordgo.MessageEmbed {
	return newEmbed("A flight has been logged!",
		field("Callsign", s.Callsign),
		field("Start Time", s.Start),
		field("End Time", discordTime(now)),
	)
}

// UpdateEmbed posts an updated flight plan when its revision has advanced.
func (pilotHandler) UpdateEmbed(p tpcgo.Pilot, existing PilotSession) (*discordgo.MessageEmbed, PilotSession, bool) {
	if p.FlightPlan == nil {
		return nil, PilotSession{}, false
	}
	storedRev, _ := strconv.Atoi(existing.RevisionId)
	if p.FlightPlan.RevisionID <= storedRev {
		return nil, PilotSession{}, false
	}

	embed := newEmbed("A flight has started! - Updated Flight Plan",
		field("Callsign", existing.Callsign),
		field("Start Time", existing.Start),
		flightPlanField("Filed Flight Plan - Updated", p.FlightPlan),
	)
	existing.applyFlightPlan(p.FlightPlan)
	return embed, existing, true
}

func (pilotHandler) MessageID(s PilotSession) string { return s.MessageId }

func (pilotHandler) SetMessageID(s PilotSession, id string) PilotSession {
	s.MessageId = id
	return s
}

// OnlineCheck performs one reconciliation pass for online pilots.
func OnlineCheck(src DataSource, st store.SessionStore[PilotSession]) {
	webhook, err := newDiscordWebhook()
	if err != nil {
		log.Printf("failed to create discord webhook: %v", err)
		return
	}
	NewTracker[tpcgo.Pilot, PilotSession](st, webhook, pilotHandler{}).Run(context.Background(), src)
}
