package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"im_project/client"
	"im_project/serv"
)

const version = "v1"

func main() {
	flag.Parse()

	root := cobra.Command{
		Use:     "chat",
		Short:   "chat demo",
		Version: version,
	}
	ctx := context.Background()

	root.AddCommand(serv.ServerStartCmd(ctx, version))
	root.AddCommand(client.NewCmd(ctx))
	// root.AddCommand()

	if err := root.Execute(); err != nil {
		logrus.WithError(err).Fatal("Could not run command")
	}
}
