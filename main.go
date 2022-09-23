package main

import (
	"dirkeeper/cmd"
	log "github.com/sirupsen/logrus"
	"time"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})
	if err := cmd.Execute(); err != nil {
		log.Errorln("Error executing main command", err)
		return
	}
}
