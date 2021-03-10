package output

import (
	"time"

	"go.uber.org/atomic"

	"github.com/sirupsen/logrus"
)

var spinner = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

const (
	timeout      = 10 * time.Second
	displaySpeed = 200 * time.Millisecond
)

type Progress interface {
	Start()
	Stop()
	Tic()
}

type progress struct {
	printer Printer
	ticChan chan struct{}
	endChan chan struct{}
	started *atomic.Bool
}

func NewProgress(printer Printer) *progress {
	return &progress{
		printer,
		make(chan struct{}),
		make(chan struct{}),
		atomic.NewBool(false),
	}
}

func (p *progress) Start() {
	if !p.started.Swap(true) {
		go p.watch()
		go p.render()
	}
}

func (p *progress) Stop() {
	if p.started.Swap(false) {
		p.endChan <- struct{}{}
		p.printer.Printf("\n")
	}
}

func (p *progress) Tic() {
	if p.started.Load() {
		p.ticChan <- struct{}{}
	}
}

func (p *progress) render() {
	i := -1
	p.printer.Printf("Scanning resources:\r")
	for {
		select {
		case <-p.endChan:
			return
		case <-time.After(displaySpeed):
			i++
			if i >= len(spinner) {
				i = 0
			}
			p.printer.Printf("Scanning resources: %s\r", spinner[i])
		}
	}
}

func (p *progress) watch() {
Loop:
	for {
		select {
		case <-p.ticChan:
			continue Loop
		case <-time.After(timeout):
			break Loop
		case <-p.endChan:
			return
		}
	}
	logrus.Debug("Progress did not receive any tic. Stopping...")
	p.endChan <- struct{}{}
}
