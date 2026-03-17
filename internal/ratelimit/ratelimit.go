package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a simple token bucket rate limiter
type Limiter struct {
	ticker *time.Ticker
	stop   chan struct{}
	tokens chan struct{}
	once   sync.Once
}

// New creates a rate limiter with rps requests per second (0 = unlimited)
func New(rps int) *Limiter {
	l := &Limiter{stop: make(chan struct{}), tokens: make(chan struct{}, rps+1)}
	if rps <= 0 {
		// Unlimited — always ready
		go func() {
			for {
				select {
				case l.tokens <- struct{}{}:
				case <-l.stop:
					return
				default:
					// keep full
					select {
					case l.tokens <- struct{}{}:
					default:
					}
					time.Sleep(time.Millisecond)
				}
			}
		}()
		return l
	}
	interval := time.Second / time.Duration(rps)
	l.ticker  = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-l.ticker.C:
				select {
				case l.tokens <- struct{}{}:
				default:
				}
			case <-l.stop:
				return
			}
		}
	}()
	return l
}

// Wait blocks until a token is available
func (l *Limiter) Wait() {
	<-l.tokens
}

// Stop shuts down the limiter
func (l *Limiter) Stop() {
	l.once.Do(func() {
		close(l.stop)
		if l.ticker != nil {
			l.ticker.Stop()
		}
	})
}
