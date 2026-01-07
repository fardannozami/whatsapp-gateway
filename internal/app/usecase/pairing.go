package usecase

import (
	"strings"
	"time"
)

const pairCodeTTL = 90 * time.Second

func pairingBackoff(err error) (time.Duration, string) {
	if err == nil {
		return 0, ""
	}

	msg := err.Error()
	if strings.Contains(msg, "rate-overlimit") || strings.Contains(msg, "429") {
		return 60 * time.Second, "rate_limited"
	}

	return 10 * time.Second, "failed"
}
