// Package syncclient allows programs to easily sync to an oscsync master.
package syncclient

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/scgolang/osc"
	"github.com/scgolang/syncosc"
	"golang.org/x/sync/errgroup"
)

// Slave is any type that can sync to an oscsync master.
// The slave's Pulse method will be invoked every time a new pulse is received
// from the oscsync master.
type Slave interface {
	Pulse(syncosc.Pulse) error
}

// Connect connects a slave to an oscsync master.
// This func blocks forever.
func Connect(ctx context.Context, slave Slave, host string) error {
	local, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return errors.Wrap(err, "creating listening address")
	}
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(syncosc.MasterPort)))
	if err != nil {
		return errors.Wrap(err, "creating listening address")
	}
	g, gctx := errgroup.WithContext(ctx)

	conn, err := osc.DialUDPContext(gctx, "udp", local, remote)
	if err != nil {
		return errors.Wrap(err, "connecting to master")
	}
	pulseChan := make(chan syncosc.Pulse, 1)

	// Start the OSC server so we receive the master's messages.
	g.Go(func() error {
		return tickerLoop(gctx, slave, pulseChan)
	})
	g.Go(func() error {
		return receivePulses(conn, pulseChan)
	})
	// Announce the slave to the master.
	portStr := strings.Split(conn.LocalAddr().String(), ":")[1]
	lport, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "parsing int from %s", portStr)
	}
	if err := conn.Send(osc.Message{
		Address: syncosc.AddressSlaveAdd,
		Arguments: osc.Arguments{
			osc.String("127.0.0.1"),
			osc.Int(lport),
		},
	}); err != nil {
		return errors.Wrap(err, "sending add-slave message")
	}
	return g.Wait()
}

func receivePulses(conn osc.Conn, pulseChan chan<- syncosc.Pulse) error {
	return conn.Serve(osc.Dispatcher{
		syncosc.AddressPulse: osc.Method(func(m osc.Message) error {
			pulse, err := syncosc.PulseFromMessage(m)
			if err != nil {
				return errors.Wrap(err, "getting pulse from message")
			}
			pulseChan <- pulse
			return nil
		}),
	})
}

func tickerLoop(ctx context.Context, slave Slave, pulseChan <-chan syncosc.Pulse) error {
	var (
		p      syncosc.Pulse
		ticker *time.Ticker
	)

	// Don't do anything until we have received the first pulse.
	// The master should send this right after the announcement is successful.
	// Announcement shouldn't take any longer than a second.
	// In fact, it shouldn't take longer than a couple milliseconds.
ExpectPulse:
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(1 * time.Second):
		return errors.New("timeout waiting for first pulse")
	case p = <-pulseChan:
	}

	ticker = time.NewTicker(syncosc.GetPulseDuration(p.Tempo))

EnterLoop:
	for range ticker.C {
		// Has there been a tempo update?
		select {
		default:
			p.Count++
			if err := slave.Pulse(p); err != nil {
				return errors.Wrap(err, "invoking slave.Pulse")
			}
			if p.Count%(syncosc.PulsesPerBar-1) == 0 {
				// We are about to hit a new bar.
				goto ExpectPulse
			}
		case <-ctx.Done():
			return nil
		case p = <-pulseChan:
			// Tempo has changed.
			ticker = time.NewTicker(syncosc.GetPulseDuration(p.Tempo))
			goto EnterLoop
		}
	}
	return nil
}
