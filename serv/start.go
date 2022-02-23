package serv

import (
	"context"
	"github.com/spf13/cobra"
)

type ServerStartOptions struct {
	id     string
	listen string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	server := NewServer(opts.id, opts.listen)
	defer server.Shutdown()
	return server.Start()
}

func ServerStartCmd(ctx context.Context, version string) *cobra.Command {
	opts := &ServerStartOptions{}
	cmd := cobra.Command{
		Use:   "chat",
		Short: "start a chat server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServerStart(ctx, opts, version)
		},
	}
	cmd.PersistentFlags().StringVarP(&opts.id, "serverid", "i", "demo", "server id")
	cmd.PersistentFlags().StringVarP(&opts.listen, "listen", "l", ":8001", "listen address")
	return &cmd
}
