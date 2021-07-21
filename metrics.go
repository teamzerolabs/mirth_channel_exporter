package main

import (
	"crypto/tls"
	"encoding/xml"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"log"
	"net/http"
)

const namespace = "mirth"
const channelStatusesApi = "/api/channels/statuses"

var (
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client = &http.Client{Transport: tr}

	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last Mirth query successful.",
		nil, nil,
	)

	channelStatuses = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "channel_status"),
		"Status of all deployed channels",
		[]string{"channel", "status"}, nil,
	)

	messagesReceived = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_received_total"),
		"How many messages have been received (per channel).",
		[]string{"channel"}, nil,
	)

	messagesFiltered = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_filtered_total"),
		"How many messages have been filtered (per channel).",
		[]string{"channel"}, nil,
	)

	messagesQueued = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_queued"),
		"How many messages are currently queued (per channel).",
		[]string{"channel"}, nil,
	)

	messagesSent = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_sent_total"),
		"How many messages have been sent (per channel).",
		[]string{"channel"}, nil,
	)

	messagesErrored = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_errored_total"),
		"How many messages have errored (per channel).",
		[]string{"channel"}, nil,
	)

	requestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    prometheus.BuildFQName(namespace, "", "request_duration"),
		Help:    "Histogram for the runtime of the metric pull from Mirth.",
		Buckets: prometheus.LinearBuckets(0.1, 0.1, 20),
	})
)

func (e *Exporter) LoadChannelStatuses() (*ChannelStatusMap, error) {
	timer := prometheus.NewTimer(requestDuration)
	defer timer.ObserveDuration()
	req, err := http.NewRequest("GET", e.mirthEndpoint+channelStatusesApi, nil)
	if err != nil {
		return nil, err
	}

	// This one line implements the authentication required for the task.
	req.SetBasicAuth(e.mirthUsername, e.mirthPassword)
	// Make request and show output.
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// fmt.Println(string(body))

	// initialize map variable
	var channelStatusMap ChannelStatusMap
	// unmarshal body byteArray into the ChannelStatusMap struct
	err = xml.Unmarshal(body, &channelStatusMap)
	if err != nil {
		return nil, err
	}

	return &channelStatusMap, nil
}

func pickMetric(status string) *prometheus.Desc {
	switch status {
	case "RECEIVED":
		return messagesReceived
	case "FILTERED":
		return messagesFiltered
	case "SENT":
		return messagesSent
	case "QUEUED":
		return messagesQueued
	case "ERROR":
		return messagesErrored
	}
	return nil
}

func (e *Exporter) AssembleMetrics(channelStatusMap *ChannelStatusMap, ch chan<- prometheus.Metric) {
	ch <- requestDuration

	for _, channel := range channelStatusMap.Channels {
		ch <- prometheus.MustNewConstMetric(
			channelStatuses, prometheus.GaugeValue, 1, channel.Name, channel.State,
		)

		for _, entry := range channel.CurrentStatistics {
			metric := pickMetric(entry.Status)
			if metric != nil {
				ch <- prometheus.MustNewConstMetric(
					metric, prometheus.GaugeValue, entry.MessageCount, channel.Name,
				)
			}
		}
	}

	log.Println("Endpoint scraped")
}

type Exporter struct {
	mirthEndpoint, mirthUsername, mirthPassword string
}

func NewExporter(mirthEndpoint string, mirthUsername string, mirthPassword string) *Exporter {
	return &Exporter{
		mirthEndpoint: mirthEndpoint,
		mirthUsername: mirthUsername,
		mirthPassword: mirthPassword,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- channelStatuses
	ch <- messagesReceived
	ch <- messagesFiltered
	ch <- messagesQueued
	ch <- messagesSent
	ch <- messagesErrored
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	channelIdStatusMap, err := e.LoadChannelStatuses()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)

	e.AssembleMetrics(channelIdStatusMap, ch)
}
