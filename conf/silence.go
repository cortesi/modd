package conf

import (
	"fmt"
	"time"
)

// A Silence (a.k.a debounce) denotes how much time should pass after last change to start the block
type Silence struct {
	LastTime time.Time
	Duration time.Duration // Silence interval duration from last change
}

func (s *Silence) String() string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("<silence for %v, last %v, %v remains>", s.Duration, s.LastTime, s.Duration - time.Since(s.LastTime))
}

// Ready checks if the Silence's timeout passed and aim it again if needed.
func (s *Silence) Ready() bool {
	if s == nil {
		return true
	}

	if s.Duration == time.Duration(0) {
		return true
	}

	if time.Since(s.LastTime) >= s.Duration {
		s.LastTime = time.Now()
		return true
	}

	return false
}
