package output

import "fmt"

var globalPrinter Printer = &VoidPrinter{}

func ChangePrinter(printer Printer) {
	globalPrinter = printer
}

func Printf(format string, args ...interface{}) {
	globalPrinter.Printf(format, args...)
}

type Printer interface {
	Printf(format string, args ...interface{})
}

type ConsolePrinter struct{}

func NewConsolePrinter() *ConsolePrinter {
	return &ConsolePrinter{}
}

func (c *ConsolePrinter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

type VoidPrinter struct{}

func (v *VoidPrinter) Printf(format string, args ...interface{}) {}
