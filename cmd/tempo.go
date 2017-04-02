// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"github.com/scgolang/syncosc"
	"github.com/spf13/cobra"
)

// tempoCmd represents the tempo command
var tempoCmd = &cobra.Command{
	Use:   "tempo",
	Short: "Change the tempo of an oscsync server.",
	Long:  `Change the tempo of an oscsync server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			flags = cmd.Flags()
			host  string
		)
		flags.StringVar(&host, "h", "127.0.0.1", "hostname of oscsync server")

		if err := flags.Parse(args); err != nil {
			return err
		}
		addr := fmt.Sprintf("%s:%d", host, syncosc.MasterPort)

		if len(flags.Args()) == 0 {
			return readTempo(addr)
		}
		raddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}
		conn, err := osc.DialUDP("udp", nil, raddr)
		if err != nil {
			return err
		}
		tempo, err := strconv.ParseFloat(flags.Args()[0], 32)
		if err != nil {
			return err
		}
		return conn.Send(osc.Message{
			Address: syncosc.AddressTempo,
			Arguments: osc.Arguments{
				osc.Float(tempo),
			},
		})
	},
}

func init() {
	RootCmd.AddCommand(tempoCmd)
}

// readTempo reads the current tempo of an oscsync server.
func readTempo(addr string) error {
	laddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	conn, err := osc.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	var (
		done    = make(chan struct{})
		errchan = make(chan error, 1)
	)
	go func() {
		if err := conn.Serve(1, osc.Dispatcher{
			"/reply": handleTempoReply(done),
		}); err != nil {
			errchan <- err
		}
		close(errchan)
	}()
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	if err := conn.SendTo(raddr, osc.Message{Address: syncosc.AddressTempo}); err != nil {
		return err
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		return errors.New("timeout waiting for tempo reply")
	}

	select {
	default:
	case err := <-errchan:
		return err
	}
	return nil
}

// handleTempoReply handles a reply to get the tempo of an oscsync server.
func handleTempoReply(done chan struct{}) osc.Method {
	return osc.Method(func(m osc.Message) error {
		defer close(done)

		if len(m.Arguments) < 1 {
			return errors.New("expected at least 1 argument to /reply")
		}
		address, err := m.Arguments[0].ReadString()
		if err != nil {
			return err
		}
		if address != syncosc.AddressTempo {
			return nil
		}
		if len(m.Arguments) < 2 {
			return errors.New("expected at least 2 arguments in tempo reply")
		}
		tempo, err := m.Arguments[1].ReadFloat32()
		if err != nil {
			return err
		}
		fmt.Printf("%f\n", tempo)
		return nil
	})
}
