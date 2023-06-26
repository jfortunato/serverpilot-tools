package apps

import (
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

var MinRuntime string
var MaxRuntime string
var CreatedAfter string
var CreatedBefore string

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [OPTIONS]",
		Aliases: []string{"ls"},
		Short:   "List apps",
		Args:    cobra.ExactArgs(2),
		//PreRunE: func(cmd *cobra.Command, args []string) error {
		//	// Validate here?
		//},
		RunE: func(cmd *cobra.Command, args []string) error {
			minRuntime := serverpilot.Runtime(MinRuntime)
			maxRuntime := serverpilot.Runtime(MaxRuntime)
			createdAfter, err := serverpilot.DateCreatedFromDate(CreatedAfter)
			if err != nil {
				return fmt.Errorf("created-after must be in the format YYYY-MM-DD")
			}
			createdBefore, err := serverpilot.DateCreatedFromDate(CreatedBefore)
			if err != nil {
				return fmt.Errorf("created-before must be in the format YYYY-MM-DD")
			}

			listApps(args[0], args[1], minRuntime, maxRuntime, createdAfter, createdBefore)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&MinRuntime, "min-runtime", "", "Only display apps with a runtime greater than or equal to the specified runtime")
	flags.StringVar(&MaxRuntime, "max-runtime", "", "Only display apps with a runtime less than or equal to the specified runtime")
	flags.StringVar(&CreatedAfter, "created-after", "", "Only display apps created after the specified date")
	flags.StringVar(&CreatedBefore, "created-before", "", "Only display apps created before the specified date")

	return cmd
}

func listApps(user, key string, minRuntime, maxRuntime serverpilot.Runtime, createdAfter, createdBefore serverpilot.DateCreated) {
	c := serverpilot.NewClient(user, key)

	apps, err := filter.FilterApps(c, minRuntime, maxRuntime, createdAfter, createdBefore)
	if err != nil {
		log.Fatalln("error while filtering apps: ", err)
	}

	printApps(apps)
}

func printApps(apps []serverpilot.App) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSERVER\tDOMAINS\tRUNTIME\tCREATED\t")
	for _, app := range apps {
		domains := strings.Join(app.Domains, ", ")
		fmt.Fprintln(w, app.Id+"\t"+app.Name+"\t"+app.Serverid+"\t"+domains+"\t"+string(app.Runtime)+"\t"+app.Datecreated.String()+"\t")
	}
	w.Flush()
}
