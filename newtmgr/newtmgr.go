/*
 Copyright 2015 Runtime Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"git-wip-us.apache.org/repos/asf/incubator-mynewt-newt/newtmgr/cli"
	"git-wip-us.apache.org/repos/asf/incubator-mynewt-newt/newtmgr/protocol"
	"git-wip-us.apache.org/repos/asf/incubator-mynewt-newt/newtmgr/transport"
	"git-wip-us.apache.org/repos/asf/incubator-mynewt-newt/util"
	"github.com/hashicorp/logutils"
	"github.com/spf13/cobra"
)

var ConnProfileName string
var LogLevel string = "WARN"

func setupLog() {
	filter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"DEBUG", "VERBOSE", "INFO",
			"WARN", "ERROR"},
		MinLevel: logutils.LogLevel(LogLevel),
		Writer:   os.Stderr,
	}

	log.SetOutput(filter)
}

func nmUsage(cmd *cobra.Command, err error) {
	if err != nil {
		sErr := err.(*util.NewtError)
		fmt.Printf("ERROR: %s\n", err.Error())
		fmt.Fprintf(os.Stderr, "[DEBUG] %s", sErr.StackTrace)
	}

	if cmd != nil {
		cmd.Help()
	}

	os.Exit(1)
}

func connProfileAddCmd(cmd *cobra.Command, args []string) {
	cpm, err := cli.NewCpMgr()
	if err != nil {
		nmUsage(cmd, err)
	}

	name := args[0]
	cp, err := cli.NewConnProfile(name)
	if err != nil {
		nmUsage(cmd, err)
	}

	for _, vdef := range args[1:] {
		s := strings.Split(vdef, "=")
		switch s[0] {
		case "name":
			cp.Name = s[1]
		case "type":
			cp.Type = s[1]
		case "connstring":
			cp.ConnString = s[1]
		default:
			nmUsage(cmd, util.NewNewtError("Unknown variable "+s[0]))
		}
	}

	if err := cpm.AddConnProfile(cp); err != nil {
		nmUsage(cmd, err)
	}

	fmt.Printf("Connection profile %s successfully added\n", name)
}

func connProfileShowCmd(cmd *cobra.Command, args []string) {
	cpm, err := cli.NewCpMgr()
	if err != nil {
		nmUsage(cmd, err)
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	cpList, err := cpm.GetConnProfileList()
	if err != nil {
		nmUsage(cmd, err)
	}

	found := false
	for _, cp := range cpList {
		// Print out the connection profile, if name is "" or name
		// matches cp.Name
		if name != "" && cp.Name != name {
			continue
		}

		if !found {
			found = true
			fmt.Printf("Connection profiles: \n")
		}
		fmt.Printf("  %s: type=%s, connstring='%s'\n", cp.Name, cp.Type,
			cp.ConnString)
	}

	if !found {
		if name == "" {
			fmt.Printf("No connection profiles found!\n")
		} else {
			fmt.Printf("No connection profiles found matching %s\n", name)
		}
	}
}

func connProfileDelCmd(cmd *cobra.Command, args []string) {
	cpm, err := cli.NewCpMgr()
	if err != nil {
		nmUsage(cmd, err)
	}

	name := args[0]

	if err := cpm.DeleteConnProfile(name); err != nil {
		nmUsage(cmd, err)
	}

	fmt.Printf("Connection profile %s successfully deleted.\n", name)
}

func connProfileCmd() *cobra.Command {
	cpCmd := &cobra.Command{
		Use:   "conn",
		Short: "Manage newtmgr connection profiles",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a newtmgr connection profile",
		Run:   connProfileAddCmd,
	}
	cpCmd.AddCommand(addCmd)

	deleCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a newtmgr connection profile",
		Run:   connProfileDelCmd,
	}
	cpCmd.AddCommand(deleCmd)

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show newtmgr connection profiles",
		Run:   connProfileShowCmd,
	}
	cpCmd.AddCommand(showCmd)

	return cpCmd
}

func echoRunCmd(cmd *cobra.Command, args []string) {
	cpm, err := cli.NewCpMgr()
	if err != nil {
		nmUsage(cmd, err)
	}

	profile, err := cpm.GetConnProfile(ConnProfileName)
	if err != nil {
		nmUsage(cmd, err)
	}

	conn, err := transport.NewConn(profile)
	if err != nil {
		nmUsage(cmd, err)
	}

	runner, err := protocol.NewCmdRunner(conn)
	if err != nil {
		nmUsage(cmd, err)
	}

	echo, err := protocol.NewEcho()
	if err != nil {
		nmUsage(cmd, err)
	}

	echo.Message = args[0]

	nmr, err := echo.EncodeWriteRequest()
	if err != nil {
		nmUsage(cmd, err)
	}

	if err := runner.WriteReq(nmr); err != nil {
		nmUsage(cmd, err)
	}

	rsp, err := runner.ReadReq()
	if err != nil {
		nmUsage(cmd, err)
	}

	ersp, err := protocol.DecodeEchoResponse(rsp.Data)
	if err != nil {
		nmUsage(cmd, err)
	}
	fmt.Println(ersp.Message)
}

func echoCmd() *cobra.Command {
	echoCmd := &cobra.Command{
		Use:   "echo",
		Short: "Send data to remote endpoint using newtmgr, and receive data back",
		Run:   echoRunCmd,
	}

	return echoCmd
}

func parseCmds() *cobra.Command {
	nmCmd := &cobra.Command{
		Use:   "newtmgr",
		Short: "Newtmgr helps you manage remote instances of the Mynewt OS.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	nmCmd.PersistentFlags().StringVarP(&ConnProfileName, "conn", "c", "",
		"connection profile to use.")

	nmCmd.PersistentFlags().StringVarP(&LogLevel, "loglevel", "l", "",
		"log level to use (default WARN.)")

	nmCmd.AddCommand(connProfileCmd())
	nmCmd.AddCommand(echoCmd())

	return nmCmd
}

func main() {
	cmd := parseCmds()
	setupLog()
	cmd.Execute()
}
