package bootstrap

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"go.uber.org/dig"

	outbox "github.com/moasq/go-b2b-starter/internal/platform/outbox"
	server "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

func Execute() {
	if err := godotenv.Load("app.env"); err != nil {
		log.Printf("Warning: Error loading app.env file: %v", err)
	}

	container := dig.New()

	InitMods(container)

	// Start the outbox dispatcher after all module subscriptions are
	// registered so dispatched events reach their listeners.
	if err := container.Invoke(func(d *outbox.Dispatcher) {
		d.Start(context.Background())
	}); err != nil {
		panic(err)
	}

	var srv server.Server

	if err := container.Invoke(func(s server.Server) {
		srv = s
	}); err != nil {
		panic(err)
	}

	srv.Start()

}
