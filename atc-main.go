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
	if err != nil {
		log.Fatalf("failed to create tpcgo session: %v", err)
	}

	for {
		log.Println("Starting ATC Online Checker Process")
		functions.ATCOnlineCheck(s)
		log.Println("ATC Online Checker Process Complete. Awaiting Datafeed Update.")
		time.Sleep(15 * time.Second)
	}
}
