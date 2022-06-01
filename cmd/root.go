package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "instance-metadata-service",
	Short: "Instance Metadata Service for Hollow ecosystem",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
