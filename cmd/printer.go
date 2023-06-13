package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/theckman/yacspin"
)

// Printer describes the terminal printing configuration.
type Printer struct {
	spinner *yacspin.Spinner
}

var printer Printer

// setup sets up the printer.
func (p *Printer) setup() {
	spinner, err := yacspin.New(
		yacspin.Config{
			Frequency:         100 * time.Millisecond,
			CharSet:           yacspin.CharSets[59],
			Message:           "Loading",
			Suffix:            " ",
			StopCharacter:     "",
			StopMessage:       "",
			StopFailCharacter: "[!] \b",
			ColorAll:          true,
			Colors:            []string{"bold", "fgYellow"},
			StopFailColors:    []string{"bold", "fgRed"},
		})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p.spinner = spinner
	p.spinner.Start()
}

// Print displays a loading spinner and a message.
func (p *Printer) Print(message string, status ...int) {
	if status != nil {
		p.Stop(message)
		os.Exit(status[0])
	}

	p.spinner.Message(message)
}

// Stop stops the spinner.
func (p *Printer) Stop(message ...string) {
	m := ""
	if message != nil {
		m = message[0]
	}

	p.spinner.StopMessage(m)
	p.spinner.Stop()
}

// Error displays an error and stops the application.
func (p *Printer) Error(message string) {
	p.spinner.StopFailMessage(message)
	p.spinner.StopFail()

	os.Exit(1)
}

// PrintError prints an error to the screen.
func PrintError(message string, err ...error) {
	if err != nil {
		message = message + ": " + err[0].Error()
	}

	printer.spinner.Start()
	printer.Error(message)
}
