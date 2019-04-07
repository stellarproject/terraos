package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "terra"
	app.Version = "2"
	app.Usage = "Terra OS management"
	app.Description = `
                                                     ___
                                                  ,o88888
                                               ,o8888888'
                         ,:o:o:oooo.        ,8O88Pd8888"
                     ,.::.::o:ooooOoOoO. ,oO8O8Pd888'"
                   ,.:.::o:ooOoOoOO8O8OOo.8OOPd8O8O"
                  , ..:.::o:ooOoOOOO8OOOOo.FdO8O8"
                 , ..:.::o:ooOoOO8O888O8O,COCOO"
                , . ..:.::o:ooOoOOOO8OOOOCOCO"
                 . ..:.::o:ooOoOoOO8O8OCCCC"o
                    . ..:.::o:ooooOoCoCCC"o:o
                    . ..:.::o:o:,cooooCo"oo:o:
                 ` + "`" + `   . . ..:.:cocoooo"'o:o:::'
                 .` + "`" + `   . ..::ccccoc"'o:o:o:::'
                :.:.    ,c:cccc"':.:.:.:.:.'
              ..:.:"'` + "`" + `::::c:"'..:.:.:.:.:.'
            ...:.'.:.::::"'    . . . . .'
           .. . ....:."' ` + "`" + `   .  . . ''
         . . . ...."'
         .. . ."'
        .
Terra OS management`
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:  "device,d",
			Usage: "select the device terra is installed to",
			Value: "/dev/sda",
		},
		cli.StringFlag{
			Name:  "fs-type",
			Usage: "set the filesystem type",
			Value: "ext4",
		},
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Commands = []cli.Command{
		osCommand,
		autoPartitionCommand,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
