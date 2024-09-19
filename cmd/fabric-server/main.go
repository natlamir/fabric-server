package main

import (
	"log"

	"github.com/yourusername/fabric-server/internal/server"
)

func main() {
	s := server.NewServer()
	log.Fatal(s.Start())
}