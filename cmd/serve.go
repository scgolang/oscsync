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
		slaves:      map[net.Addr]struct{}{},

		tempoChan: make(chan float32, 8),
	}
	return srv, nil
}

// HandleSlaveAdd handles the OSC message to add a slave.
func (srv *Server) HandleSlaveAdd(m osc.Message) error {
	addr, err := readUDPAddr(m)
	if err != nil {
		return errors.Wrap(err, "getting addr from osc message")
	}
	srv.slaveAdd <- addr
	return nil
}

// HandleSlaveRemove handles the OSC message to remove a slave.
func (srv *Server) HandleSlaveRemove(m osc.Message) error {
	addr, err := readUDPAddr(m)
	if err != nil {
		return errors.Wrap(err, "getting addr from osc message")
	}
	srv.slaveRemove <- addr
	return nil
}

// HandleTempo handles tempo updates.
func (srv *Server) HandleTempo(m osc.Message) error {
	if len(m.Arguments) == 0 {
		return srv.conn.SendTo(m.Sender, osc.Message{
			Address: "/reply",
			Arguments: osc.Arguments{
				osc.String(syncosc.AddressTempo),
				osc.Float(srv.tempo),
			},
		})
	}
	tempo, err := m.Arguments[0].ReadFloat32()
	if err != nil {
		return errors.Wrap(err, "reading float argument")
	}
	srv.tempoChan <- tempo
	return nil
}

// incrPulse increments the pulse and broadcasts to all slaves.
func (srv *Server) incrPulse(newTempo, oldTempo float32, newSlave net.Addr) error {
	srv.pulse++

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
	if newSlave != nil {
		if err := srv.sendPulse(srv.pulse, []net.Addr{newSlave}, newTempo); err != nil {
			return errors.Wrap(err, "sending pulse")
		}
	}
	return nil
}

// Main is the main loop of the server.
func (srv *Server) Main(ctx context.Context) error {
	srv.ticker = time.NewTicker(syncosc.GetPulseDuration(srv.tempo))

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
			srv.ticker = time.NewTicker(syncosc.GetPulseDuration(newTempo))
		}
		if err := srv.incrPulse(newTempo, srv.tempo, newSlave); err != nil {
			return errors.Wrap(err, "incrementing pulse")
		}
		if srv.tempo != newTempo {
			srv.tempo = newTempo
			goto EnterLoop // re-enter the loop since we've changed the ticker
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
		return oscsrv.Serve(2, osc.Dispatcher{
			syncosc.AddressTempo:       osc.Method(srv.HandleTempo),
			syncosc.AddressSlaveAdd:    osc.Method(srv.HandleSlaveAdd),
			syncosc.AddressSlaveRemove: osc.Method(srv.HandleSlaveRemove),
		})
	})
	g.Go(func() error {
		return srv.Main(ctx)
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

// readUDPAddr reads a host/port from an osc message and returns it as a net.Addr
func readUDPAddr(m osc.Message) (net.Addr, error) {
	if expected, got := 2, len(m.Arguments); expected != got {
		return nil, errors.Errorf("expected %d arguments, got %d", expected, got)
	}
	host, err := m.Arguments[0].ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "reading host")
	}
	port, err := m.Arguments[1].ReadInt32()
	if err != nil {
		return nil, errors.Wrap(err, "reading port")
	}
	return net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(int(port))))
}
