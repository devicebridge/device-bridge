package main

import (
	"context"

	"github.com/devicebridge/device-bridge/internal/source/scanner"
)

type hidRuntime interface {
	Input() <-chan scanner.Input
	Run(context.Context) error
	Close()
}
