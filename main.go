package main

import (
	"crypto/tls"
	"encoding/xml"
	"flag"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type ChannelStatusMap struct {
	XMLName  xml.Name        `xml:"list"`
	Channels []ChannelStatus `xml:"dashboardStatus"`
}
type ChannelStatus struct {
	XMLName            xml.Name                       `xml:"dashboardStatus"`
	ChannelId          string                         `xml:"channelId"`
	Name               string                         `xml:"name"`
	State              string                         `xml:"state"`
	CurrentStatistics  []ChannelStatusStatisticsEntry `xml:"statistics>entry"`
	LifetimeStatistics []ChannelStatusStatisticsEntry `xml:"lifetimeStatistics>entry"`

	/*
		<statistics class="linked-hash-map">
		  <entry>
			<com.mirth.connect.donkey.model.message.Status>RECEIVED</com.mirth.connect.donkey.model.message.Status>
			<long>70681</long>
		  </entry>
		  <entry>
			<com.mirth.connect.donkey.model.message.Status>FILTERED</com.mirth.connect.donkey.model.message.Status>
			<long>0</long>
		  </entry>
		  <entry>
			<com.mirth.connect.donkey.model.message.Status>SENT</com.mirth.connect.donkey.model.message.Status>
			<long>67139</long>
		  </entry>
		  <entry>
			<com.mirth.connect.donkey.model.message.Status>ERROR</com.mirth.connect.donkey.model.message.Status>
			<long>3542</long>
		  </entry>
		</statistics>
	*/
}

type ChannelStatusStatisticsEntry struct {
	Status       string  `xml:"com.mirth.connect.donkey.model.message.Status"`
	MessageCount float64 `xml:"long"`
}

const namespace = "mirth"
const channelStatusesApi = "/api/channels/statuses"

var (
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	listenAddress = flag.String("web.listen-address", ":9141",
		"Address to listen on for telemetry")
	metricsPath = flag.String("web.telemetry-path", "/metrics",
		"Path under which to expose metrics")

	// Metrics
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
			channelStatuses, prometheus.UntypedValue, 1, channel.Name, channel.State,
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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, assume env variables are set.")
	}

	flag.Parse()

	mirthEndpoint := os.Getenv("MIRTH_ENDPOINT")
	mirthUsername := os.Getenv("MIRTH_USERNAME")
	mirthPassword := os.Getenv("MIRTH_PASSWORD")

	exporter := NewExporter(mirthEndpoint, mirthUsername, mirthPassword)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Mirth Channel Exporter</title></head>
             <body>
             <h1>Mirth Channel Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Println("Listening on ", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
