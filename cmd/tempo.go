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
		if len(flags.Args()) < 1 {
			return errors.New("usage: oscsync [OPTIONS] tempo BPM")
		}
		tempo, err := strconv.ParseFloat(flags.Args()[0], 32)
		if err != nil {
			return err
		}
		addr := fmt.Sprintf("%s:%d", host, syncosc.MasterPort)
		raddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}
		conn, err := osc.DialUDP("udp", nil, raddr)
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
