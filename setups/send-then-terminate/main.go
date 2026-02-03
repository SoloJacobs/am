package main

import (
	"time"

	"github.com/SoloJacobs/am/orchestrate"
)

func main() {
	_, err := orchestrate.StartLocalCluster("send-then-terminate", 1)
	if err != nil {
		panic(err)
	}
	_, err = orchestrate.StartReceiver()
	if err != nil {
		panic(err)
	}
	time.Sleep(3 * time.Second)
	now := time.Now()
	hostPayload := []orchestrate.Alert{{
		Labels: map[string]string{
			"alertname": "HostDown",
		},
		Annotations: map[string]string{
			"summary":     "Host is down.",
			"description": "Cause by cluster outage.",
		},
		StartsAt: now,
		EndsAt:   now.Add(10 * time.Minute),
	}}
	orchestrate.SendAlert(hostPayload, 9093)
	orchestrate.KillProcByPort(9093)
	_, err = orchestrate.StartLocalCluster("send-then-terminate", 1)
	if err != nil {
		panic(err)
	}
	select {}
}
