package main

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

type RedisStore struct {
	CID      string `redis:"cid"`
	Callsign string `redis:"callsign"`
	Start    string `redis:"start"`
}

func OnlineCheck(s *tpcgo.Session, err error) {

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

	for _, v := range o.Pilots {
		dfmap[strconv.Itoa(v.CID)] = v
	}

	for _, uu := range u {
		var usr RedisStore
		// CHeck Redis for cid
		err := rdb.HGetAll(ctx, "online:"+strconv.Itoa(uu.VATSIMCid)).Scan(usr)
		if err != nil {
			continue
		} else {
			if _, found := dfmap[usr.CID]; found {
				continue
			} else {
				fmt.Println("deleting new member: " + strconv.Itoa(uu.VATSIMCid))
				_, dgerr := d.WebhookExecute(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), false, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{
						{
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
									Value: time.Now().String(),
								},
							},
							Color: 3651327,
							Footer: &discordgo.MessageEmbedFooter{
								Text:    "Made by the TPC Tech Team",
								IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
							}}},
					AvatarURL: "https://cdn.thepilotclub.org/fcp/tpc%20logo.png",
					Username:  "TPC Flight Tracking",
				})
				if dgerr != nil {
					log.Fatal(dgerr)
				}
				rdb.Del(ctx, "online:"+strconv.Itoa(uu.VATSIMCid))
			}
		}

		if value, found := dfmap[strconv.Itoa(uu.VATSIMCid)]; found {
			opp, _ := rdb.HGet(ctx, "online:"+strconv.Itoa(uu.VATSIMCid), "cid").Result()
			if len(opp) > 0 {
				continue
			} else {
				var v = value.(tpcgo.Pilot)
				if v.FlightPlan != nil {
					if strings.Contains(v.FlightPlan.Remarks, "CALLSIGN PILOT CLUB") {
						_, reerr := rdb.HSet(ctx, "online:"+strconv.Itoa(uu.VATSIMCid), []string{
							"cid", strconv.Itoa(v.CID),
							"callsign", v.Callsign,
							"start", v.LogonTime,
						}).Result()
						if reerr != nil {
							fmt.Println(err)
						}
						_, dgerr := d.WebhookExecute(os.Getenv("WEBHOOK_ID"), os.Getenv("WEBHOOK_TOKEN"), false, &discordgo.WebhookParams{
							Embeds: []*discordgo.MessageEmbed{
								{
									Title: "A flight has started!",
									Fields: []*discordgo.MessageEmbedField{
										{
											Name:  "Callsign",
											Value: usr.Callsign,
										},
										{
											Name:  "Start Time",
											Value: usr.Start,
										},
									},
									Color: 3651327,
									Footer: &discordgo.MessageEmbedFooter{
										Text:    "Made by the TPC Tech Team",
										IconURL: "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png",
									}}},
							AvatarURL: "https://cdn.thepilotclub.org/fcp/tpc%20logo.png",
							Username:  "TPC Flight Tracking",
						})
						if dgerr != nil {
							log.Fatal(dgerr)
						}
					}
				}
			}
		}

	}
	err = rdb.Close()
	if err != nil {
		return
	}

}
