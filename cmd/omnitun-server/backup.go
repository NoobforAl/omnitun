package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newBackupCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "backup <file.tar.gz|->",
		Short: "Save the state, keys and certificates to a file (\"-\" = stdout); it holds every secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var w io.Writer = os.Stdout
			if args[0] != "-" {
				f, err := os.OpenFile(args[0], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
				if err != nil {
					return err
				}
				defer f.Close()
				w = f
			}
			if err := o.manager().Backup(w); err != nil {
				if args[0] != "-" {
					os.Remove(args[0])
				}
				return err
			}
			if args[0] != "-" {
				fmt.Fprintf(os.Stderr, "backup written to %s (keep it private)\n", args[0])
			}
			return nil
		},
	}
}

func newRestoreCmd(o *options) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <file.tar.gz|->",
		Short: "Restore a backup into the data directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var r io.Reader = os.Stdin
			if args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			if err := o.manager().Restore(r, force); err != nil {
				return err
			}
			fmt.Printf("restored into %s; start the server with: omnitun-server run\n", o.dataDir)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing server")
	return cmd
}
