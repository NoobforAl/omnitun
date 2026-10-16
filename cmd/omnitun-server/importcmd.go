package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/NoobforAl/omnitun/server/manager"
)

func newImportCmd(o *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Move users and keys from another server",
	}
	var opts manager.ImportXrayOptions
	xray := &cobra.Command{
		Use:   "xray <config.json|->",
		Short: "Import an Xray/V2Ray server config: users, UUIDs, passwords and the Reality identity",
		Long: `Reads the inbounds of an Xray or V2Ray server config and creates the same users
here. UUIDs, Trojan passwords and (for a new server) the VLESS+Reality key, short
ID and site are kept, so the links your users already have keep working once
your domain points at this server.

With no server yet, give --host to create it. On an existing server only the
users are added; --adopt-reality also takes over the imported Reality identity.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var raw []byte
			var err error
			if args[0] == "-" {
				raw, err = io.ReadAll(os.Stdin)
			} else {
				raw, err = os.ReadFile(args[0])
			}
			if err != nil {
				return err
			}
			res, err := o.manager().ImportXray(string(raw), opts)
			if err != nil {
				return err
			}
			return o.print(res, func() {
				if res.CreatedServer {
					fmt.Printf("Created the server in %s.\n", o.dataDir)
				}
				fmt.Printf("Added %d user(s): %v\n", len(res.UsersAdded), res.UsersAdded)
				if len(res.UsersSkipped) > 0 {
					fmt.Printf("Skipped (already exist): %v\n", res.UsersSkipped)
				}
				if res.AdoptedReality {
					fmt.Println("Took over the VLESS+Reality identity: existing client links keep working.")
				}
				for _, n := range res.Notes {
					fmt.Println("note:", n)
				}
				fmt.Println("\nShow a user's new links with: omnitun-server link <name>")
			})
		},
	}
	xray.Flags().StringVar(&opts.Host, "host", "", "public IP or domain (needed when there is no server yet)")
	xray.Flags().StringVar(&opts.Name, "name", "", "server name for a newly created server")
	xray.Flags().BoolVar(&opts.AdoptReality, "adopt-reality", false, "replace this server's Reality identity with the imported one")
	cmd.AddCommand(xray)
	return cmd
}
