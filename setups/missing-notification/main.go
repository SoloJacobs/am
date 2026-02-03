package main

import (
	"time"

	"github.com/SoloJacobs/am/orchestrate"
)

func main() {
	_, err := orchestrate.StartLocalCluster("missing-notification", 2)
	if err != nil {
		panic(err)
	}
	_, err = orchestrate.StartReceiver()
	if err != nil {
		panic(err)
	}
	time.Sleep(3 * time.Second)
	for {
		now := time.Now()
		clusterPayload := []orchestrate.Alert{
			{
				Labels: map[string]string{
					"alertname": "ClusterDown",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary":     "Cluster is down.",
					"description": "HostDown does not matter.",
				},
				StartsAt: now,
				EndsAt:   now.Add(10 * time.Minute),
			},
		}
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
		orchestrate.SendAlert(clusterPayload, 9093)
		orchestrate.SendAlert(hostPayload, 9093)
		orchestrate.SendAlert(hostPayload, 9095)
		time.Sleep(1 * time.Minute)
	}
}
