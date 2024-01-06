package main

import "encoding/xml"

type ChannelStatistics struct {
	XMLName   xml.Name `xml:"channelStatistics"`
	ServerId  string   `xml:"ServerId"`
	ChannelId string   `xml:"channelId"`
	Received  float64  `xml:"received"`
	Sent      float64  `xml:"sent"`
	Error     float64  `xml:"error"`
	Queued    float64  `xml:"queued"`
}

/*
  <channelStatistics>
    <serverId>6d555cac-1671-481f-abae-7e1e791eb2d5</serverId>
    <channelId>d7ecfa9c-4f73-4280-b8bc-ced3a154b1ea</channelId>
    <received>39</received>
    <sent>39</sent>
    <error>0</error>
    <filtered>0</filtered>
    <queued>0</queued>
  </channelStatistics>
*/

type ChannelStatisticsMap struct {
	XMLName  xml.Name            `xml:"list"`
	Channels []ChannelStatistics `xml:"channelStatistics"`
}

type ChannelStatus struct {
	XMLName               xml.Name                       `xml:"dashboardStatus"`
	ChannelId             string                         `xml:"channelId"`
	Name                  string                         `xml:"name"`
	State                 string                         `xml:"state"`
	DeployedRevisionDelta float64                        `xml:"deployedRevisionDelta"`
	CurrentStatistics     []ChannelStatusStatisticsEntry `xml:"statistics>entry"`
	LifetimeStatistics    []ChannelStatusStatisticsEntry `xml:"lifetimeStatistics>entry"`

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

type ChannelStatusMap struct {
	XMLName  xml.Name        `xml:"list"`
	Channels []ChannelStatus `xml:"dashboardStatus"`
}

type ChannelStatusStatisticsEntry struct {
	Status       string  `xml:"com.mirth.connect.donkey.model.message.Status"`
	MessageCount float64 `xml:"long"`
}
