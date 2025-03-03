package main

import (
	"os"
	"testing"
)

func Test_main(t *testing.T) {
	t.Run("main sanity check", func(t *testing.T) {
		os.Args = []string{"taskctl run unknown"}
		taskctlRootCmd, stop := cmdSetUp()
		defer stop()
		if err := taskctlRootCmd.Execute(); err == nil {
			t.Error("got nil wanted error")
		}
	})
}
