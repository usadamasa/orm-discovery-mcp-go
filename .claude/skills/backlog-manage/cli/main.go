package main

import (
	"fmt"
	"os"

	"github.com/usadamasa/backlog-cli/cmd"
)

func main() {
	args := os.Args[1:]

	// Parse global --dir flag
	dir := ".backlog"
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--dir" {
			dir = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	subcmd := args[0]
	subArgs := args[1:]

	var err error
	switch subcmd {
	case "add":
		err = cmd.RunAdd(dir, subArgs)
	case "complete":
		err = cmd.RunComplete(dir, subArgs)
	case "list":
		err = cmd.RunList(dir, subArgs)
	case "promote":
		err = cmd.RunPromote(dir, subArgs)
	case "audit":
		err = cmd.RunAudit(dir, subArgs)
	case "retrospective":
		err = cmd.RunRetrospective(dir, subArgs)
	case "regenerate-md":
		err = cmd.RunRegenerateMD(dir)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: backlog-cli [--dir DIR] <command> [args...]

Commands:
  add <task|idea|issue>  Add a new item
  complete <id>          Mark an item as done/resolved
  list [--type TYPE]     List active items
  promote                Promote an idea to task/issue
  audit [--last] [--failures]  Show eval audit log
  audit log-entry              Write audit log entry
  retrospective [--last N]     Analyze audit history
  regenerate-md          Regenerate markdown summaries`)
}
