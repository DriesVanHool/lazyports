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
		return runKill(context.Background(), args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
}

func runKill(ctx context.Context, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("usage: lazyports kill PORT [--force|--graceful-only]")
	}

	port, err := strconv.Atoi(args[0])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %q", args[0])
	}

	opts := ports.TerminateOptions{}
	for _, arg := range args[1:] {
		switch arg {
		case "--force":
			opts.Force = true
		case "--graceful-only":
			opts.GracefulOnly = true
		default:
			return fmt.Errorf("unknown kill option %q", arg)
		}
	}
	if opts.Force && opts.GracefulOnly {
		return errors.New("--force and --graceful-only are mutually exclusive")
	}

	terminated, forced, err := ports.TerminateByPort(ctx, port, opts)
	if terminated > 0 {
		switch {
		case opts.Force:
			fmt.Printf("Force killed %d process(es) bound to port %d\n", terminated, port)
		case opts.GracefulOnly:
			fmt.Printf("Gracefully terminated %d process(es) bound to port %d\n", terminated, port)
		case forced > 0:
			fmt.Printf("Terminated %d process(es) bound to port %d (%d required force kill fallback)\n", terminated, port, forced)
		default:
			fmt.Printf("Gracefully terminated %d process(es) bound to port %d\n", terminated, port)
		}
	}
	return err
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

	fmt.Fprintln(w, "PORT\tPROCESS\tPID\tPROTOCOL\tSTATE\tLOCAL\tREMOTE\tKIND")
	for _, entry := range entries {
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State, formatLocal(entry), formatRemote(entry), entry.Kind)
	}

	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  lazyports                           Launch the interactive TUI")
	fmt.Println("  lazyports list                      Print listening port bindings")
	fmt.Println("  lazyports list --all                Print listeners and active connections")
	fmt.Println("  lazyports kill PORT                 Try graceful terminate, then force kill if needed")
	fmt.Println("  lazyports kill PORT --graceful-only Only attempt graceful termination")
	fmt.Println("  lazyports kill PORT --force         Force kill immediately")
	fmt.Println("  lazyports version                   Print version information")
}

func printVersion() {
	fmt.Printf("lazyports %s\n", version)
	fmt.Printf("commit: %s\n", commit)
	fmt.Printf("built: %s\n", date)
}

func formatLocal(entry ports.Entry) string {
	if entry.LocalAddress == "" {
		return fmt.Sprintf(":%d", entry.Port)
	}
	return fmt.Sprintf("%s:%d", entry.LocalAddress, entry.Port)
}

func formatRemote(entry ports.Entry) string {
	if entry.RemoteAddress == "" {
		return "-"
	}
	if entry.RemotePort > 0 {
		return fmt.Sprintf("%s:%d", entry.RemoteAddress, entry.RemotePort)
	}
	return entry.RemoteAddress
}
