package main

import "github.com/jfortunato/serverpilot-tools/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(cmd.VersionDetails{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
}
