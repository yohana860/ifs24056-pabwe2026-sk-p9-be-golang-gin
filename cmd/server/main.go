// Command server menjalankan Delcom Blog (Gin REST API + frontend React yang di-embed).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

// Variabel ini dapat diganti pada test agar main() dapat diuji tanpa PostgreSQL / port sungguhan.
var (
	exit    = os.Exit
	connect = database.Connect
	listen  = listenAndServe
)

func listenAndServe(addr string, handler http.Handler) error {
	log.Printf("Listening and serving HTTP on %s", addr)
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

func run() error { return server.Run(config.LoadConfig(), connect, listen) }

func main() {
	if err := run(); err != nil {
		log.Printf("server berhenti: %v", err)
		exit(1)
	}
}
