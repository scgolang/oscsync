package main

import (
	"flag"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
)

func main() {
	var (
		config = Config{}
		fs     = flag.NewFlagSet("oscsync", flag.ExitOnError)
	)
	fs.Float64Var(&config.Tempo, "t", 120, "tempo in BPM")

	app, err := NewApp(config)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// App holds the state of the application.
type App struct {
	Config
}

// NewApp creates a new app.
func NewApp(config Config) (*App, error) {
	return &App{Config: config}, nil
}

// Config represents the app's configuration
type Config struct {
	Tempo float64 // bpm
}

// Run runs the application.
func (app *App) Run() error {
	return nil
}

// Tempo handles tempo updates.
func (app *App) Tempo(m osc.Message) error {
	if len(m.Arguments) < 1 {
		return errors.New("expected at least one argument")
	}
	tempo, err := m.Arguments[0].ReadFloat32()
	if err != nil {
		return errors.Wrap(err, "reading tempo")
	}
	app.Config.Tempo = float64(tempo)
	// TODO: set sleep period
	return nil
}

// getPulseNS gets the length of a pulse in nanoseconds.
func getPulseNS(bpm float32) time.Duration {
	return time.Duration(float32(6e10) / bpm)
}
