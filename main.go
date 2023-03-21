// Package main the entrypoint to the metadata service application
package main

//go:generate sqlboiler crdb

import (
	_ "go.uber.org/automaxprocs"

	"go.hollow.sh/metadataservice/cmd"
)

func main() {
	cmd.Execute()
}
