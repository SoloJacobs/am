package main

import (
	"time"

	"github.com/SoloJacobs/am/orchestrate"
)

func createPayload(now time.Time) []orchestrate.Alert {
	return []orchestrate.Alert{
		{
			Labels: map[string]string{
				"alertname": "ConstantNag",
			},
			Annotations: map[string]string{
				"summary":     "The system is still broken",
				"description": "Nag all day.",
			},
			StartsAt: now,
			EndsAt:   now.Add(10 * time.Minute),
		},
	}
}

func main() {
	_, err := orchestrate.StartLocalCluster("eternal-nagging")
	if err != nil {
		panic(err)
	}
	_, err = orchestrate.StartReceiver()
	if err != nil {
		panic(err)
	}
	time.Sleep(3 * time.Second)
	for {
		t := time.Now()
		orchestrate.SendAlert(createPayload(t), 9093)
		orchestrate.SendAlert(createPayload(t), 9095)
		orchestrate.SendAlert(createPayload(t), 9097)
		time.Sleep(1 * time.Minute)
	}
}
