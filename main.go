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
}

type ChannelStatusStatistics struct {
	Entries []ChannelStatusStatisticsEntry `xml:"entry"`
	/*
		The stats returned from status API
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
	messageCounts = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "messages_total"),
		"How many messages have been processed per channel and status.",
		[]string{"channel", "status"}, nil,
	)
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
	ch <- messageCounts
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

func (e *Exporter) AssembleMetrics(channelStatusMap *ChannelStatusMap, ch chan<- prometheus.Metric) {

	for _, channel := range channelStatusMap.Channels {
		ch <- prometheus.MustNewConstMetric(
			channelStatuses, prometheus.UntypedValue, 1, channel.Name, channel.State,
		)

		for _, entry := range channel.CurrentStatistics {
			ch <- prometheus.MustNewConstMetric(
				messageCounts, prometheus.CounterValue, entry.MessageCount, channel.Name, entry.Status,
			)
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
