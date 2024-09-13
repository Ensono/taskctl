package cmd

import (
	"fmt"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/spf13/cobra"
)

func newValidateCmd(parentCmd *cobra.Command, configFunc func() (*config.Config, error)) {
	c := &cobra.Command{
		Use:   "validate",
		Short: `validates config file`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := configFunc()
			if err != nil {
				return err
			}
			fmt.Fprintln(ChannelOut, "file is valid")
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return nil // postRunReset()
		},
	}
	parentCmd.AddCommand(c)
}
