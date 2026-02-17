package main

import (
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/the-pilot-club/online-checker/functions"
	"github.com/the-pilot-club/tpcgo"
)

func main() {

	s, err := tpcgo.NewSession(tpcgo.SessionConfig{
		FCPEnv: "production", // Leaving Blank due to it not being needed
	})

	for {
		log.Println("Starting ATC Online Checker Process")
		functions.ATCOnlineCheck(s, err)
		log.Println("ATC Online Checker Process Complete. Awaiting Datafeed Update.")
		time.Sleep(15 * time.Second)
	}
}
