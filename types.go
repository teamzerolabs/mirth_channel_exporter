package main

import "encoding/xml"

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
