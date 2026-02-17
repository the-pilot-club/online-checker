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
	"github.com/redis/go-redis/v9"
	"github.com/the-pilot-club/tpcgo"

	_ "github.com/joho/godotenv/autoload"
)

type ATCRedisStore struct {
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

func ATCOnlineCheck(s *tpcgo.Session, err error) {

	dbnum, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL") + ":6379",
		DB:       dbnum,
		Protocol: 3,
	})
	ctx := context.Background()

	d, err := discordgo.New("")
	//var embed *discordgo.MessageEmbed

	u, err := s.GetAllFCPUsersCID()
	if err != nil {
		fmt.Println("Error getting users: ", err)
		return
	}
	o, err := s.GetVatsimDataFeed()
	if err != nil {
		fmt.Println("Error getting vatsim data", err)
		return
	}

	var dfmap = make(map[string]interface{})

	for _, v := range o.Controllers {
		dfmap[strconv.Itoa(v.CID)] = v
	}

	for _, uu := range u {

		if value, found := dfmap[strconv.Itoa(uu.VATSIMCid)]; found {
			opp, _ := rdb.HGet(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid), "cid").Result()
			if len(opp) > 0 {
				var usru RedisStore
				var vv = value.(tpcgo.Pilot)
				_ = rdb.HGetAll(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid)).Scan(&usru)
				if vv.FlightPlan != nil {
					if strconv.Itoa(vv.FlightPlan.RevisionID) > usru.RevisionId {
						embedd := &discordgo.MessageEmbed{
							Title: "A flight has started! - Updated Flight Plan",
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:  "Callsign",
									Value: usru.Callsign,
								},
								{
									Name:  "Start Time",
									Value: usru.Start,
								},
							},
							Color: 3651327,
							Footer: &discordgo.MessageEmbedFooter{
								Text:    "Made by the TPC Tech Team",
								IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
							}}
						embedd.Fields = append(embedd.Fields, &discordgo.MessageEmbedField{
							Name:  "Filed Flight Plan - Updated",
							Value: fmt.Sprintf("> A/C: %v\n> DEP: %v\n> ARR: %v\n> Route: %v\n> Remarks: %v", vv.FlightPlan.AircraftShort, vv.FlightPlan.Departure, vv.FlightPlan.Arrival, vv.FlightPlan.Route, vv.FlightPlan.Remarks),
						})
						_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), usru.MessageId, &discordgo.WebhookEdit{
							Embeds: &[]*discordgo.MessageEmbed{embedd},
						})
						if dgerr != nil {
							log.Fatal(dgerr)
						}
					}
				}
				continue
			} else {
				var v = value.(tpcgo.Pilot)
				if strings.Contains(v.Callsign, "TPC") {
					embedstart := &discordgo.MessageEmbed{
						Title: "A flight has started!",
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Callsign",
								Value: v.Callsign,
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
					if v.FlightPlan != nil {
						embedstart.Fields = append(embedstart.Fields, &discordgo.MessageEmbedField{
							Name:  "Filed Flight Plan",
							Value: fmt.Sprintf("> A/C: %v\n> DEP: %v\n> ARR: %v\n> Route: %v\n> Remarks: %v", v.FlightPlan.AircraftShort, v.FlightPlan.Departure, v.FlightPlan.Arrival, v.FlightPlan.Route, v.FlightPlan.Remarks),
						})
					}
					w, dgerr := d.WebhookExecute(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), true, &discordgo.WebhookParams{
						Embeds:    []*discordgo.MessageEmbed{embedstart},
						AvatarURL: "https://cdn.thepilotclub.org/fcp/tpc%20logo.png",
						Username:  "TPC Flight Tracking",
					})
					if dgerr != nil {
						log.Fatal(dgerr)
					}
					store := []string{
						"cid", strconv.Itoa(v.CID),
						"callsign", v.Callsign,
						"start", fmt.Sprintf("<t:%d:f>", time.Now().Unix()),
						"message_id", w.ID,
					}
					if v.FlightPlan != nil {
						store = append(store, []string{
							"aircraft_short", v.FlightPlan.AircraftShort,
							"departure", v.FlightPlan.Departure,
							"arrival", v.FlightPlan.Arrival,
							"route", v.FlightPlan.Route,
							"remarks", v.FlightPlan.Remarks,
							"revision_id", strconv.Itoa(v.FlightPlan.RevisionID),
						}...)
					}
					_, reerr := rdb.HSet(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid), store).Result()
					if reerr != nil {
						fmt.Println(err)
					}
				}
			}
		} else {
			var usr RedisStore
			_ = rdb.HGetAll(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid)).Scan(&usr)
			if usr.CID != "" {
				embedd := &discordgo.MessageEmbed{
					Title: "A flight has been logged!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Callsign",
							Value: usr.Callsign,
						},
						{
							Name:  "Start Time",
							Value: usr.Start,
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
				_, dgerr := d.WebhookMessageEdit(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), usr.MessageId, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embedd},
				})
				if dgerr != nil {
					log.Fatal(dgerr)
				}
				rdb.Del(ctx, "online-atc:"+strconv.Itoa(uu.VATSIMCid))
			}
		}

	}
	err = rdb.Close()
	if err != nil {
		return
	}

}
