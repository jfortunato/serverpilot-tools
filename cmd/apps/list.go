package apps

import (
	"encoding/json"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/spf13/cobra"
	"log"
	"os"
	"text/tabwriter"
)

var MinRuntime string
var MaxRuntime string

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [OPTIONS]",
		Aliases: []string{"ls"},
		Short:   "List apps",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			minRuntime := serverpilot.Runtime(MinRuntime)
			maxRuntime := serverpilot.Runtime(MaxRuntime)

			listApps(args[0], args[1], minRuntime, maxRuntime)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&MinRuntime, "min-runtime", "", "Only display apps with a runtime greater than or equal to the specified runtime")
	flags.StringVar(&MaxRuntime, "max-runtime", "", "Only display apps with a runtime less than or equal to the specified runtime")

	return cmd
}

func listApps(user, key string, minRuntime, maxRuntime serverpilot.Runtime) {
	c := serverpilot.NewClient(user, key)

	apps, err := filter.FilterApps(c, minRuntime, maxRuntime)
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
	fmt.Fprintln(w, "ID\tNAME\tRUNTIME\t")
	for _, app := range apps {
		fmt.Fprintln(w, app.Id+"\t"+app.Name+"\t"+string(app.Runtime)+"\t")
	}
	w.Flush()
}
