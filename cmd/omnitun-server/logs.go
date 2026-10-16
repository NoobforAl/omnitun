package main

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newLogsCmd(o *options) *cobra.Command {
	var lines int
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show the server log (what `run` reported: reloads, quota blocks, warnings, errors)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := o.manager()
			out, err := m.Logs(lines)
			if err != nil {
				return err
			}
			if o.json && !follow {
				return o.print(out, func() {})
			}
			for _, l := range out {
				fmt.Println(l)
			}
			if !follow {
				return nil
			}
			// Follow: print what is new, until Ctrl+C.
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			last := ""
			if len(out) > 0 {
				last = out[len(out)-1]
			}
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-sig:
					return nil
				case <-tick.C:
				}
				cur, err := m.Logs(500)
				if err != nil {
					return err
				}
				start := 0
				for i, c := range slices.Backward(cur) {
					if c == last {
						start = i + 1
						break
					}
				}
				for _, l := range cur[start:] {
					fmt.Println(l)
					last = l
				}
			}
		},
	}
	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "how many lines to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "keep printing new lines")
	return cmd
}
