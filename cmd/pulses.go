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
	"fmt"

	"github.com/scgolang/syncclient"
	"github.com/scgolang/syncosc"
	"github.com/spf13/cobra"
)

// pulsesCmd represents the pulses command
var pulsesCmd = &cobra.Command{
	Use:   "pulses",
	Short: "Display pulses from oscsync on stdout",
	Long:  `Display pulses from oscsync on stdout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			ctx   = context.Background()
			flags = cmd.Flags()
			host  = "127.0.0.1"
			n     = 1
			ps    = pulseSlave{}
		)
		flags.StringVar(&host, "h", host, "oscsync master host name")
		flags.IntVar(&n, "n", n, "Only display every n pulses (default is 1, i.e. every pulse)")
		return syncclient.Connect(ctx, ps, host)
	},
}

func init() {
	RootCmd.AddCommand(pulsesCmd)
}

type pulseSlave struct{}

// Pulse pulses the slave.
func (ps pulseSlave) Pulse(p syncosc.Pulse) error {
	fmt.Printf("%d\n", p.Count)
	return nil
}
