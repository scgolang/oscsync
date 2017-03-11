// Copyright © 2017 Brian Sorahan <bsorahan@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"github.com/scgolang/syncosc"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an oscsync server",
	Long:  `Start an oscsync server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			config = ServerConfig{}
			flags  = cmd.Flags()
		)
		flags.StringVar(&config.host, "h", "0.0.0.0", "listen addr")
		flags.IntVar(&config.port, "p", 5776, "listen port")
		flags.Float32Var(&config.tempo, "t", 120, "tempo in bpm")

		if err := flags.Parse(args); err != nil {
			return errors.Wrap(err, "parsing flags")
		}
		srv, err := NewServer(config)
		if err != nil {
			return errors.Wrap(err, "creationg server")
		}
		return errors.Wrap(srv.Run(), "running server")
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
}

// Server runs an oscsync server.
type Server struct {
	ServerConfig

	ctx       context.Context
	pulse     uint64
	tempoChan chan float32
}

// NewServer creates a new oscsync server.
func NewServer(config ServerConfig) (*Server, error) {
	srv := &Server{
		ServerConfig: config,

		ctx:       context.Background(),
		tempoChan: make(chan float32, 1),
	}
	return srv, nil
}

// HandleTempo handles tempo updates.
func (srv *Server) HandleTempo(m osc.Message) error {
	if expected, got := 1, len(m.Arguments); expected != got {
		return errors.Errorf("expected %d argument(s), got %d", expected, got)
	}
	tempo, err := m.Arguments[0].ReadFloat32()
	if err != nil {
		return errors.Wrap(err, "reading float argument")
	}
	srv.tempoChan <- tempo
	return nil
}

func (srv *Server) loop(ctx context.Context) error {
	ticker := time.NewTicker(srv.getPulseNS())

	for {
		select {
		case <-ticker.C:
			if srv.pulse++; srv.pulse%96 == 0 {
				// Send bar
			}
		case tempo := <-srv.tempoChan:
			ticker.Stop()
			srv.tempo = tempo
			ticker = time.NewTicker(srv.getPulseNS())
		}
	}
	return nil
}

// Run runs an oscsync server.
func (srv *Server) Run() error {
	// Run the osc server.
	g, ctx := errgroup.WithContext(srv.ctx)

	laddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(srv.host, strconv.Itoa(srv.port)))
	if err != nil {
		return errors.Wrap(err, "resolving listen address")
	}
	oscsrv, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		return errors.Wrap(err, "creating OSC server")
	}
	g.Go(func() error {
		return oscsrv.Serve(osc.Dispatcher{
			syncosc.AddressTempo: osc.Method(srv.HandleTempo),
		})
	})
	g.Go(func() error {
		return srv.loop(ctx)
	})
	return g.Wait()
}

// ServerConfig contains configurationn for an oscsync server.
type ServerConfig struct {
	host  string
	port  int
	tempo float32
}

func (config ServerConfig) getPulseNS() time.Duration {
	return time.Duration(float32(25e8) / config.tempo)
}
