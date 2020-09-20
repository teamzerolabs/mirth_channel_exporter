// A minimal example of how to include Prometheus instrumentation.
package main

import (
	"crypto/tls"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/*
<map>
  <entry>
    <string>101af57f-f26c-40d3-86a3-309e74b93512</string>
    <string>Send-Email-Notification</string>
  </entry>
</map>
*/
type ChannelIdNameMap struct {
	XMLName xml.Name       `xml:"map"`
	Entries []ChannelEntry `xml:"entry"`
}
type ChannelEntry struct {
	XMLName xml.Name `xml:"entry"`
	Values  []string `xml:"string"`
}

/*
<list>
  <channelStatistics>
    <serverId>c5e6a736-0e88-46a7-bf32-5b4908c4d859</serverId>
    <channelId>101af57f-f26c-40d3-86a3-309e74b93512</channelId>
    <received>0</received>
    <sent>0</sent>
    <error>0</error>
    <filtered>0</filtered>
    <queued>0</queued>
  </channelStatistics>
</list>
*/
type ChannelStatsList struct {
	XMLName  xml.Name       `xml:"list"`
	Channels []ChannelStats `xml:"channelStatistics"`
}
type ChannelStats struct {
	XMLName   xml.Name `xml:"channelStatistics"`
	ServerId  string   `xml:"serverId"`
	ChannelId string   `xml:"channelId"`
	Received  string   `xml:"received"`
	Sent      string   `xml:"sent"`
	Error     string   `xml:"error"`
	Filtered  string   `xml:"filtered"`
	Queued    string   `xml:"queued"`
}

const namespace = "mirth"
const channelIdNameApi = "/api/channels/idsAndNames"
const channelStatsApi = "/api/channels/statistics"

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
	ch <- messagesReceived
	ch <- messagesFiltered
	ch <- messagesQueued
	ch <- messagesSent
	ch <- messagesErrored
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	channelIdNameMap, err := e.LoadChannelIdNameMap()
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

	e.HitMirthRestApisAndUpdateMetrics(channelIdNameMap, ch)
}

func (e *Exporter) LoadChannelIdNameMap() (map[string]string, error) {
	// Create the map of channel id to names
	channelIdNameMap := make(map[string]string)

	req, err := http.NewRequest("GET", e.mirthEndpoint+channelIdNameApi, nil)
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

	// we initialize our array
	var channelIdNameMapXML ChannelIdNameMap
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(body, &channelIdNameMapXML)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(channelIdNameMapXML.Entries); i++ {
		channelIdNameMap[channelIdNameMapXML.Entries[i].Values[0]] = channelIdNameMapXML.Entries[i].Values[1]
	}

	return channelIdNameMap, nil
}

func (e *Exporter) HitMirthRestApisAndUpdateMetrics(channelIdNameMap map[string]string, ch chan<- prometheus.Metric) {
	// Load channel stats
	req, err := http.NewRequest("GET", e.mirthEndpoint+channelStatsApi, nil)
	if err != nil {
		log.Fatal(err)
	}

	// This one line implements the authentication required for the task.
	req.SetBasicAuth(e.mirthUsername, e.mirthPassword)
	// Make request and show output.
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(string(body))

	// we initialize our array
	var channelStatsList ChannelStatsList
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(body, &channelStatsList)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(channelStatsList.Channels); i++ {
		channelName := channelIdNameMap[channelStatsList.Channels[i].ChannelId]

		channelReceived, _ := strconv.ParseFloat(channelStatsList.Channels[i].Received, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesReceived, prometheus.GaugeValue, channelReceived, channelName,
		)

		channelSent, _ := strconv.ParseFloat(channelStatsList.Channels[i].Sent, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesSent, prometheus.GaugeValue, channelSent, channelName,
		)

		channelError, _ := strconv.ParseFloat(channelStatsList.Channels[i].Error, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesErrored, prometheus.GaugeValue, channelError, channelName,
		)

		channelFiltered, _ := strconv.ParseFloat(channelStatsList.Channels[i].Filtered, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesFiltered, prometheus.GaugeValue, channelFiltered, channelName,
		)

		channelQueued, _ := strconv.ParseFloat(channelStatsList.Channels[i].Queued, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesQueued, prometheus.GaugeValue, channelQueued, channelName,
		)
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
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
