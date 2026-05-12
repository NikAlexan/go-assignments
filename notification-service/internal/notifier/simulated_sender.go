package notifier

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"
)

// SimulatedSender logs notifications instead of sending real emails.
// It simulates network latency and random transient failures to allow testing retry logic.
type SimulatedSender struct{}

func NewSimulatedSender() *SimulatedSender {
	return &SimulatedSender{}
}

func (s *SimulatedSender) Send(_ context.Context, to, subject, body string) error {
	// Simulate network latency
	latency := time.Duration(100+rand.Intn(400)) * time.Millisecond
	time.Sleep(latency)

	// Simulate ~30% failure rate
	if rand.Float32() < 0.30 {
		return errors.New("simulated: provider temporarily unavailable")
	}

	log.Printf("[SIMULATED] email to=%s subject=%q body=%q (latency=%s)", to, subject, body, latency)
	return nil
}
