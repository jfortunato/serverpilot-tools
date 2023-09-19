package progressbar

import "github.com/schollz/progressbar/v3"

// Ticker is simply used as an indicator that the progress bar should increment
type Ticker interface {
	Tick()
}

// ProgressBar is a wrapper around the progressbar library, and implements the Ticker interface
type ProgressBar struct {
	b *progressbar.ProgressBar
}

func NewProgressBar(max int, description string) *ProgressBar {
	return &ProgressBar{b: progressbar.Default(int64(max), description)}
}

func (p *ProgressBar) Tick() {
	p.b.Add(1)
}

func (p *ProgressBar) Clear() {
	p.b.Clear()
}

func (p *ProgressBar) Finish() {
	p.b.Finish()
}
