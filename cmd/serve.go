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

	conn osc.Conn
	ctx  context.Context

	pulse uint64

	slaves      map[net.Addr]struct{}
	slaveAdd    chan net.Addr
	slaveRemove chan net.Addr

	tempoChan chan float32
	ticker    *time.Ticker
}

// NewServer creates a new oscsync server.
func NewServer(config ServerConfig) (*Server, error) {
	srv := &Server{
		ServerConfig: config,

		ctx: context.Background(),

		slaveAdd:    make(chan net.Addr, 8),
		slaveRemove: make(chan net.Addr, 8),

		tempoChan: make(chan float32, 8),
	}
	return srv, nil
}

// HandleSlaveAdd handles the OSC message to add a slave.
func (srv *Server) HandleSlaveAdd(m osc.Message) error {
	// TODO
	return nil
}

// HandleSlaveRemove handles the OSC message to remove a slave.
func (srv *Server) HandleSlaveRemove(m osc.Message) error {
	// TODO
	return nil
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

// incrPulse increments the pulse and may also broadcast the current pulse to all slaves.
// pulses are broadcast to slaves if there is a tempo update, or if we have started a new bar.
func (srv *Server) incrPulse(newTempo, oldTempo float32, newSlave net.Addr) error {
	if srv.pulse++; srv.pulse%96 == 0 || newTempo != oldTempo {
		var (
			i      = 0
			slaves = make([]net.Addr, len(srv.slaves))
		)
		for slave := range srv.slaves {
			slaves[i] = slave
			i++
		}
		if err := srv.sendPulse(srv.pulse, slaves, newTempo); err != nil {
			return errors.Wrap(err, "sending pulse")
		}
	} else if newSlave != nil {
		if err := srv.sendPulse(srv.pulse, []net.Addr{newSlave}, newTempo); err != nil {
			return errors.Wrap(err, "sending pulse")
		}
	}
	return nil
}

// loop is the main loop of the server.
func (srv *Server) loop(ctx context.Context) error {
	srv.ticker = time.NewTicker(getPulseNS(srv.tempo))

EnterLoop:
	for range srv.ticker.C {
		var (
			newSlave net.Addr
			newTempo = srv.tempo
		)
		select {
		default:
		case newSlave = <-srv.slaveAdd:
			srv.slaves[newSlave] = struct{}{}
		case slave := <-srv.slaveRemove:
			delete(srv.slaves, slave)
		case newTempo = <-srv.tempoChan:
			srv.ticker.Stop()
			srv.ticker = time.NewTicker(getPulseNS(newTempo))
		}
		if err := srv.incrPulse(newTempo, srv.tempo, newSlave); err != nil {
			return errors.Wrap(err, "incrementing pulse")
		}
		if srv.tempo != newTempo {
			srv.tempo = newTempo
			goto EnterLoop
		}
	}
	return nil
}

// Run runs an oscsync server.
func (srv *Server) Run() error {
	// Run the osc server.
	g, ctx := errgroup.WithContext(srv.ctx)

	laddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(srv.host, strconv.Itoa(syncosc.MasterPort)))
	if err != nil {
		return errors.Wrap(err, "resolving listen address")
	}
	oscsrv, err := osc.ListenUDPContext(ctx, "udp", laddr)
	if err != nil {
		return errors.Wrap(err, "creating OSC server")
	}
	srv.conn = oscsrv

	g.Go(func() error {
		return oscsrv.Serve(osc.Dispatcher{
			syncosc.AddressTempo:       osc.Method(srv.HandleTempo),
			syncosc.AddressSlaveAdd:    osc.Method(srv.HandleSlaveAdd),
			syncosc.AddressSlaveRemove: osc.Method(srv.HandleSlaveRemove),
		})
	})
	g.Go(func() error {
		return srv.loop(ctx)
	})
	return g.Wait()
}

// sendPulse sends a pulse message to addr.
func (srv *Server) sendPulse(pulse uint64, slaves []net.Addr, tempo float32) error {
	if srv.conn == nil {
		return errors.New("OSC connection has not been initialized")
	}
	for _, slave := range slaves {
		if err := srv.conn.SendTo(slave, osc.Message{
			Address: syncosc.AddressPulse,
			Arguments: osc.Arguments{
				osc.Float(tempo),
				osc.Int(int32(pulse)),
			},
		}); err != nil {
			return errors.Wrapf(err, "sending pulse message to %s", slave)
		}
	}
	return nil
}

// ServerConfig contains configurationn for an oscsync server.
type ServerConfig struct {
	host  string
	tempo float32
}

// getPulseNS converts the tempo in bpm to a time.Duration
// callers are responsible for making concurrent access safe.
func getPulseNS(tempo float32) time.Duration {
	if tempo == 0 {
		return time.Duration(0)
	}
	return time.Duration(float32(25e8) / tempo)
}
