package main

import (
	"encoding/json"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"log"
	"os"
	"text/tabwriter"
)

func main() {
	user := os.Args[1]
	key := os.Args[2]
	minRuntime := serverpilot.Runtime(os.Args[3])

	c := serverpilot.NewClient(user, key)

	apps, err := filter.FilterApps(c, minRuntime, "")
	if err != nil {
		log.Fatalln("error while filtering apps: ", err)
	}

	//prettyPrint(apps)
	printApps(apps)
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "  ")
	log.Println(string(s))
}

func printApps(apps []serverpilot.App) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tRuntime\t")
	for _, app := range apps {
		fmt.Fprintln(w, app.Id+"\t"+app.Name+"\t"+string(app.Runtime)+"\t")
	}
	w.Flush()
}
