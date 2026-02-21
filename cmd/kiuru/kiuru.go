package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/lukahietala/kiuru/internal/ansi"
	"github.com/lukahietala/kiuru/internal/editor"
	"golang.org/x/term"
)

func main() {
	debugPtr := flag.Bool("debug", false, "show runtime stats")
	flag.Parse()

	input := editor.NewTermInput(os.Stdin)
	renderer := editor.NewTermRenderer(os.Stdout, int(os.Stdout.Fd()))
	e := editor.NewEditor(input, renderer, *debugPtr)

	// Load all buffers
	args := flag.Args()
	if len(args) > 0 {
		for _, arg := range args {
			buf, _ := editor.NewBuffer(arg, false, true)
			e.AddBuffer(buf)
		}
	} else {
		buf, _ := editor.NewBuffer("", true, false)
		e.AddBuffer(buf)
	}

	// Terminal setup
	os.Stdout.WriteString(ansi.AltBufferOn + ansi.ClearScreen + ansi.CursorHome)
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		os.Stdout.WriteString(ansi.AltBufferOff + ansi.ClearScreen + ansi.CursorHome)

		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Crashed?!? Recovering...\n")
			fmt.Fprintf(os.Stderr, "Error: %v\n", r)
			debug.PrintStack()
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			e.UpdateSize()
			e.Draw()
		}
	}()

	e.Draw()

	for {
		key, err := input.ReadKey()
		if err != nil {
			break
		}

		e.ProcessKey(key)

		for input.Buffered() > 0 {
			key, err := input.ReadKey()
			if err != nil {
				break
			}
			e.ProcessKey(key)
		}

		if e.ShouldQuit() {
			break
		}

		e.Draw()
	}
}
