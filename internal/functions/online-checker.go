package functions

import (
	"context"
	"fmt"
	"log"
	"os"
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

func OnlineCheck(s *tpcgo.Session, st store.SessionStore[PilotSession]) {

	ctx := context.Background()

	d, err := discordgo.New("")
	if err != nil {
		log.Printf("failed to create discord client: %v", err)
		return
	}

	u, err := s.GetAllFCPUsersCID()
	if err != nil {
		log.Println("Error getting users: ", err)
		return
	}
	o, err := s.GetVatsimDataFeed()
	if err != nil {
		log.Println("Error getting vatsim data", err)
		return
	}

	var dfmap = make(map[string]tpcgo.Pilot)

	for _, v := range o.Pilots {
		dfmap[strconv.Itoa(v.CID)] = v
	}

	for _, uu := range u {

		cid := strconv.Itoa(uu.VATSIMCid)

		existing, found, err := st.Get(ctx, cid)
		if err != nil {
			log.Printf("store get failed for cid %d: %v", uu.VATSIMCid, err)
			continue
		}

		pilot, online := dfmap[cid]
		if !online {
			// The pilot is no longer connected: close out a tracked flight.
			if found {
				embedd := &discordgo.MessageEmbed{
					Title: "A flight has been logged!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Callsign",
							Value: existing.Callsign,
						},
						{
							Name:  "Start Time",
							Value: existing.Start,
						},
						{
							Name:  "End Time",
							Value: fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
						},
					},
					Color: 3651327,
					Footer: &discordgo.MessageEmbedFooter{
						Text:    "Made by the TPC Tech Team",
						IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
					}}
				_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), existing.MessageId, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embedd},
				})
				if dgerr != nil {
					log.Printf("webhook edit (end) failed for cid %d: %v", uu.VATSIMCid, dgerr)
					continue
				}
				if derr := st.Delete(ctx, cid); derr != nil {
					log.Printf("store delete failed for cid %d: %v", uu.VATSIMCid, derr)
				}
			}
			continue
		}

		if found {
			// Flight already announced: post an update if the filed flight
			// plan has been revised.
			if pilot.FlightPlan != nil {
				storedRev, _ := strconv.Atoi(existing.RevisionId)
				if pilot.FlightPlan.RevisionID > storedRev {
					embedd := &discordgo.MessageEmbed{
						Title: "A flight has started! - Updated Flight Plan",
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Callsign",
								Value: existing.Callsign,
							},
							{
								Name:  "Start Time",
								Value: existing.Start,
							},
						},
						Color: 3651327,
						Footer: &discordgo.MessageEmbedFooter{
							Text:    "Made by the TPC Tech Team",
							IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
						}}
					embedd.Fields = append(embedd.Fields, &discordgo.MessageEmbedField{
						Name:  "Filed Flight Plan - Updated",
						Value: fmt.Sprintf("> A/C: %v\n> DEP: %v\n> ARR: %v\n> Route: %v\n> Remarks: %v", pilot.FlightPlan.AircraftShort, pilot.FlightPlan.Departure, pilot.FlightPlan.Arrival, pilot.FlightPlan.Route, pilot.FlightPlan.Remarks),
					})
					_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), existing.MessageId, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{embedd},
					})
					if dgerr != nil {
						log.Printf("webhook edit failed for cid %d: %v", uu.VATSIMCid, dgerr)
						continue
					}
					existing.AircraftShort = pilot.FlightPlan.AircraftShort
					existing.Departure = pilot.FlightPlan.Departure
					existing.Arrival = pilot.FlightPlan.Arrival
					existing.Route = pilot.FlightPlan.Route
					existing.Remarks = pilot.FlightPlan.Remarks
					existing.RevisionId = strconv.Itoa(pilot.FlightPlan.RevisionID)
					if serr := st.Set(ctx, cid, existing); serr != nil {
						log.Printf("store set (update) failed for cid %d: %v", uu.VATSIMCid, serr)
					}
				}
			}
			continue
		}

		// New flight: announce it and start tracking.
		if !strings.Contains(pilot.Callsign, "TPC") {
			continue
		}

		embedstart := &discordgo.MessageEmbed{
			Title: "A flight has started!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Callsign",
					Value: pilot.Callsign,
				},
				{
					Name:  "Start Time",
					Value: fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
				},
			},
			Color: 3651327,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Made by the TPC Tech Team",
				IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
			}}
		if pilot.FlightPlan != nil {
			embedstart.Fields = append(embedstart.Fields, &discordgo.MessageEmbedField{
				Name:  "Filed Flight Plan",
				Value: fmt.Sprintf("> A/C: %v\n> DEP: %v\n> ARR: %v\n> Route: %v\n> Remarks: %v", pilot.FlightPlan.AircraftShort, pilot.FlightPlan.Departure, pilot.FlightPlan.Arrival, pilot.FlightPlan.Route, pilot.FlightPlan.Remarks),
			})
		}
		w, dgerr := d.WebhookExecute(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), true, &discordgo.WebhookParams{
			Embeds:    []*discordgo.MessageEmbed{embedstart},
			AvatarURL: "https://cdn.thepilotclub.org/fcp/tpc%20logo.png",
			Username:  "TPC Flight Tracking",
		})
		if dgerr != nil {
			log.Printf("webhook execute failed for cid %d: %v", uu.VATSIMCid, dgerr)
			continue
		}

		session := PilotSession{
			CID:       strconv.Itoa(pilot.CID),
			Callsign:  pilot.Callsign,
			Start:     fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
			MessageId: w.ID,
		}
		if pilot.FlightPlan != nil {
			session.AircraftShort = pilot.FlightPlan.AircraftShort
			session.Departure = pilot.FlightPlan.Departure
			session.Arrival = pilot.FlightPlan.Arrival
			session.Route = pilot.FlightPlan.Route
			session.Remarks = pilot.FlightPlan.Remarks
			session.RevisionId = strconv.Itoa(pilot.FlightPlan.RevisionID)
		}
		if serr := st.Set(ctx, cid, session); serr != nil {
			log.Printf("store set failed for cid %d: %v", uu.VATSIMCid, serr)
		}
	}
}
