# Mirth Channel Exporter

Export [Mirth Connect](https://en.wikipedia.org/wiki/Mirth_Connect) channel
statistics to [Prometheus](https://prometheus.io).

Metrics are retrieved using the Mirth Connect REST API. This was tested in versions 3.7.1 and newer. It is generally expected to work in any MC release 3.4.0 and greater but has not been explicitly tested in all releases. Test cases are welcome.

To run it:

    go build
    ./mirth_channel_exporter [flags]

## Exported Metrics
| Metric | Description | Labels |
| ------ | ------- | ------ |
| mirth_up | Was the last Mirth CLI query successful | |
| mirth_request_duration | Histogram for the runtime of the metric pull from Mirth | |
| mirth_channel_status | Status of all deployed channels | channel, status |
| mirth_messages_received_total | How many messages have been received | channel |
| mirth_messages_filtered_total  | How many messages have been filtered | channel |
| mirth_messages_queued | How many messages are currently queued | channel |
| mirth_messages_sent_total  | How many messages have been sent | channel |
| mirth_messages_errored_total  | How many messages have errored | channel |

```
# HELP mirth_channel_status
# TYPE mirth_channel_status gauge
mirth_channel_status{channel="foo", status="STARTED"} 1
mirth_channel_status{channel="bar", status="PAUSED"} 1

# HELP mirth_request_duration Histogram for the runtime of the metric pull from Mirth.
# TYPE mirth_request_duration histogram
mirth_request_duration_bucket{le="0.1"} 0
mirth_request_duration_bucket{le="0.2"} 0
mirth_request_duration_bucket{le="0.30000000000000004"} 1
...
mirth_request_duration_bucket{le="2.0000000000000004"} 5
mirth_request_duration_bucket{le="+Inf"} 5

# HELP mirth_messages_errored_total How many messages have errored (per channel).
# TYPE mirth_messages_errored_total gauge
mirth_messages_errored_total{channel="foo"} 0
mirth_messages_errored_total{channel="bar"} 2

# HELP mirth_messages_filtered_total How many messages have been filtered (per channel).
# TYPE mirth_messages_filtered_total gauge
mirth_messages_filtered_total{channel="foo"} 0
mirth_messages_filtered_total{channel="bar"} 193

# HELP mirth_messages_queued How many messages are currently queued (per channel).
# TYPE mirth_messages_queued gauge
mirth_messages_queued{channel="foo"} 0
mirth_messages_queued{channel="bar"} 0

# HELP mirth_messages_received_total How many messages have been received (per channel).
# TYPE mirth_messages_received_total gauge
mirth_messages_received_total{channel="foo"} 6.3965406e+07
mirth_messages_received_total{channel="bar"} 387

# HELP mirth_messages_sent_total How many messages have been sent (per channel).
# TYPE mirth_messages_sent_total gauge
mirth_messages_sent_total{channel="foo"} 1.21855264e+08
mirth_messages_sent_total{channel="bar"} 964

# HELP mirth_up Was the last Mirth query successful.
# TYPE mirth_up gauge
mirth_up 1
```

## Flags
    ./mirth_channel_exporter --help

| Flag | Description | Default |
| ---- | ----------- | ------- |
| log.level | Logging level | `info` |
| web.listen-address | Address to listen on for telemetry | `:9141` |
| web.telemetry-path | Path under which to expose metrics | `/metrics` |

## Env Variables

Use a .env file in the local folder, or /etc/sysconfig/mirth_channel_exporter
```
MIRTH_ENDPOINT=https://mirth-connect.yourcompane.com
MIRTH_USERNAME=admin
MIRTH_PASSWORD=admin
```

## Notice

This exporter is inspired by the [consul_exporter](https://github.com/prometheus/consul_exporter)
and has some common code. Any new code here is Copyright &copy; 2020 TeamZero, Inc. See the included
LICENSE file for terms and conditions.
