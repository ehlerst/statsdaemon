package udp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/raintank/statsdaemon/common"
	"github.com/raintank/statsdaemon/out"
	reuse "github.com/libp2p/go-reuseport"
	log "github.com/sirupsen/logrus"
    reuse "github.com/libp2p/go-reuseport"
	"net"
	"strconv"
)

const (
	MaxUdpPacketSize = 65535
)

// ParseLine turns a line into a *Metric (or not) and returns an error if the line was invalid.
// note that *Metric can be nil when the line was valid (if the line was empty)
// input format: key:value|modifier[|@samplerate]
func ParseLine(line []byte) (metric *common.Metric, err error) {
	if len(line) == 0 {
		return nil, nil
	}
	parts := bytes.SplitN(bytes.TrimSpace(line), []byte(":"), 2)
	if len(parts) != 2 {
		return nil, errors.New("bad amount of colons")
	}
	if bytes.Contains(parts[1], []byte(":")) {
		return nil, errors.New("bad amount of colons")
	}
	bucket := parts[0]
	if len(bucket) == 0 {
		return nil, errors.New("key zero len")
	}
	parts = bytes.SplitN(parts[1], []byte("|"), 3)
	if len(parts) < 2 {
		return nil, errors.New("bad amount of pipes")
	}
	modifier := string(parts[1])
	if modifier != "g" && modifier != "c" && modifier != "ms" {
		return nil, errors.New("unsupported metric type")
	}
	sampleRate := float64(1)
	if len(parts) == 3 {
		if parts[2][0] != byte('@') {
			return nil, errors.New("invalid sampling")
		}
		var err error
		sampleRate, err = strconv.ParseFloat(string(parts[2])[1:], 32)
		if err != nil {
			return nil, err
		}
	}
	value, err := strconv.ParseFloat(string(parts[0]), 64)
	if err != nil {
		return nil, err
	}
	metric = &common.Metric{
		Bucket:   string(bucket),
		Value:    value,
		Modifier: modifier,
		Sampling: float32(sampleRate),
	}
	return metric, nil
}

// ParseMessage turns byte data into a slice of metric pointers
// note that it creates "invalid line" metrics itself, upon invalid lines,
// which will get passed on and aggregated along with the other metrics
func ParseMessage(data []byte, prefix_internal string, output *out.Output, parse parseLineFunc) (metrics []*common.Metric) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		metric, err := parse(line)
		if err != nil {
			// data will be repurposed by the udpListener
			report_line := make([]byte, len(line), len(line))
			copy(report_line, line)
			output.Invalid_lines.Broadcast <- report_line
			metric = &common.Metric{
				Bucket:   fmt.Sprintf("%smtype_is_count.type_is_invalid_line.unit_is_Err", prefix_internal),
				Value:    float64(1),
				Modifier: "c",
				Sampling: float32(1),
			}
		} else {
			// data will be repurposed by the udpListener
			report_line := make([]byte, len(line), len(line))
			copy(report_line, line)
			output.Valid_lines.Broadcast <- report_line
		}
		if metric != nil {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

type parseLineFunc func(line []byte) (metric *common.Metric, err error)

func StatsListener(listen_addr, prefix_internal string, output *out.Output) {
	Listener(listen_addr, prefix_internal, output, ParseLine2)
}


// Listener receives packets from the udp buffer, parses them and feeds both the Metrics channel
// as well as the metricAmounts channel
func Listener(listen_addr, prefix_internal string, output *out.Output, parse parseLineFunc) {
	address, err := net.ResolveUDPAddr("udp", listen_addr)
	if err != nil {
		log.Fatalf("ERROR: Cannot resolve '%s' - %s", listen_addr, err)
	}

	// listen on the same port
	listener, err := reuse.ListenPacket("udp", listen_addr)

	if err != nil {
		log.Fatalf("ERROR: ListenUDP - %s", err)
	}
	defer listener.Close()
	log.Infof("listening on %s", address)

	message := make([]byte, MaxUdpPacketSize)
	for {
		n, remaddr, err := listener.ReadFrom(message)
		if err != nil {
			log.Errorf("ERROR: reading UDP packet from %+v - %s", remaddr, err)
			continue
		}
		metrics := ParseMessage(message[:n], prefix_internal, output, parse)
		output.Metrics <- metrics
		output.MetricAmounts <- metrics
	}
}
