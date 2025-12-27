* Setup a Prometheus instance, which sends the same active alert to a cluster of Alertmanagers.
* Expand the logging in Alertmanager to write the notifications to a file.
* Write  test, which can use the file to replay the events without the Prometheus instance.
