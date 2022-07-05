// Copyright (c) 2022. SailPoint Technologies, Inc. All rights reserved.

package event

// stats is a struct representing the statistics emitted by librdkafka.
// Other fields may be added to this struct as needed.
//
// Complete structure can be referenced in https://github.com/edenhill/librdkafka/blob/master/STATISTICS.md
type stats struct {
	Topics map[string]struct {
		Partitions map[string]struct {
			ConsumerLag int64 `json:"consumer_lag"`
		} `json:"partitions"`
	} `json:"topics"`
}

// getConsumerLag returns consumer lag of specified topic's partition
func (s stats) getConsumerLag(topic string, partition string) int64 {
	if t, ok := s.Topics[topic]; ok {
		if p, ok := t.Partitions[partition]; ok {
			return p.ConsumerLag
		}
	}

	return -1
}
