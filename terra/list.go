package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli"
)

var listCommand = cli.Command{
	Name:  "list",
	Usage: "list installed vhosts",
	Action: func(clix *cli.Context) error {
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		fmt.Fprint(w, "ID\n")

		dirs, err := ioutil.ReadDir(clix.GlobalString("root"))
		if err != nil {
			if os.IsNotExist(err) {
				return w.Flush()
			}
			return err
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			fmt.Fprintf(w, "%s\n", d.Name())
		}
		return w.Flush()
	},
}
