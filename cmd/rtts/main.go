package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/charmbracelet/huh"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result, err := cli.Run(ctx, os.Stdout)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) || ctx.Err() != nil {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(1)
	}

	if result.Failed > 0 {
		os.Exit(1)
	}
}
