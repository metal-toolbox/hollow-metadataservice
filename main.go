// Package main the entrypoint to the metadata service application
package main

//go:generate sqlboiler crdb

import (
	"go.hollow.sh/metadataservice/cmd"
)

func main() {
	cmd.Execute()
}
