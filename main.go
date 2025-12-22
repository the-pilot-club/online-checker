package main

import (
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/the-pilot-club/tpcgo"
)

func main() {

	s, err := tpcgo.NewSession(tpcgo.SessionConfig{
		FCPEnv: "production", // Leaving Blank due to it not being needed
	})

	for {
		log.Println("Starting Online Checker Process")
		OnlineCheck(s, err)
		log.Println("Online Checker Process Complete. Awaiting Datafeed Update.")
		time.Sleep(15 * time.Second)
	}
}
