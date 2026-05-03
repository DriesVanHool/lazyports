package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/DriesVanHool/lazyports/internal/ports"
	"github.com/DriesVanHool/lazyports/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return ui.Run()
	}

	switch args[0] {
	case "list":
		return runList(context.Background(), args[1:])
	case "version", "--version", "-v":
		printVersion()
		return nil
	case "kill":
		if len(args) != 2 {
			return errors.New("usage: lazyports kill <port>")
		}

		port, err := strconv.Atoi(args[1])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %q", args[1])
		}

		killed, err := ports.KillByPort(context.Background(), port)
		if err != nil {
			return err
		}

		fmt.Printf("Killed %d process(es) bound to port %d\n", killed, port)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
}

func runList(ctx context.Context, args []string) error {
	options := ports.ListOptions{}
	for _, arg := range args {
		switch arg {
		case "--all":
			options.IncludeConnections = true
		default:
			return fmt.Errorf("unknown list option %q", arg)
		}
	}

	entries, err := ports.List(ctx, options)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "PORT\tPROCESS\tPID\tPROTOCOL\tSTATE\tKIND")
	for _, entry := range entries {
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State, entry.Kind)
	}

	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  lazyports            Launch the interactive TUI")
	fmt.Println("  lazyports list       Print listening port bindings")
	fmt.Println("  lazyports list --all Print listeners and active connections")
	fmt.Println("  lazyports kill PORT  Kill all processes bound to PORT")
	fmt.Println("  lazyports version    Print version information")
}

func printVersion() {
	fmt.Printf("lazyports %s\n", version)
	fmt.Printf("commit: %s\n", commit)
	fmt.Printf("built: %s\n", date)
}
