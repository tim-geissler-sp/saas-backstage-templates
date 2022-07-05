package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/grafana-tools/sdk"
)

func intP(i int) *int {
	return &i
}

func uintP(i uint) *uint {
	return &i
}

func boolP(b bool) *bool {
	return &b
}

func addGraphPanelDefaults(graph *sdk.Panel) {
	var datasource = "prometheus"

	graph.Datasource = &datasource
	graph.Lines = true
	graph.Span = 0

	graph.GridPos.H = intP(8)
	graph.GridPos.W = intP(12)
	graph.GridPos.X = intP(12)
	graph.GridPos.Y = intP(0)

	graph.GraphPanel.AliasColors = map[string]interface{}{}
	graph.GraphPanel.Dashes = boolP(false)
	graph.GraphPanel.DashLength = uintP(10)
	graph.GraphPanel.Fill = 1
	graph.GraphPanel.Legend.Show = true
	graph.GraphPanel.Linewidth = 1
	//graph.GraphPanel.NullPointMode = "null"
	graph.GraphPanel.NullPointMode = "null as zero"
	graph.GraphPanel.Pointradius = 2
	graph.GraphPanel.SeriesOverrides = []sdk.SeriesOverride{}
	graph.GraphPanel.SpaceLength = uintP(10)
	graph.GraphPanel.Thresholds = []sdk.Threshold{}
	graph.GraphPanel.Tooltip.Shared = true
	graph.GraphPanel.Tooltip.ValueType = "individual"
	graph.GraphPanel.Xaxis.Show = true
	graph.GraphPanel.XAxis = false
	graph.GraphPanel.Yaxes = []sdk.Axis{
		{
			Format:  "short",
			Label:   "",
			LogBase: 1,
			Max:     nil,
			Min:     nil,
			Show:    true,
		},
		{
			Format:  "short",
			Label:   "",
			LogBase: 1,
			Max:     nil,
			Min:     nil,
			Show:    true,
		},
	}
	graph.GraphPanel.YAxis = false
}

// For each op graph 90th percentile latency and average latency.
func buildHistVecLatencyGraph(panelName string, metricName string) *sdk.Panel {
	graph := sdk.NewGraph(panelName)
	addGraphPanelDefaults(graph)

	graph.GraphPanel.Yaxes[0].Format = "ms"
	graph.AddTarget(&sdk.Target{
		RefID:        "A",
		LegendFormat: "{{op}} 90th Percentile",
		Expr:         fmt.Sprintf("histogram_quantile(0.9, sum(rate(%s_bucket[$__rate_interval])) by (op,le))", metricName),
	})
	graph.AddTarget(&sdk.Target{
		RefID:        "B",
		LegendFormat: "{{op}} Average",
		Expr:         fmt.Sprintf("(sum(rate(%s_sum[$__rate_interval])) by (op)) / (sum(rate(%s_count[$__rate_interval])) by (op))", metricName, metricName),
	})
	return graph
}

// For each op graph ops/second
func buildHistVecRateGraph(panelName string, metricName string) *sdk.Panel {
	graph := sdk.NewGraph(panelName)
	addGraphPanelDefaults(graph)

	graph.GraphPanel.Yaxes[0].Format = "ops"
	graph.AddTarget(&sdk.Target{
		RefID:        "A",
		LegendFormat: "{{op}} / sec",
		Expr:         fmt.Sprintf("sum(rate(%s_count[5m])) by (op)", metricName),
	})
	return graph
}

// Graph 90th percentile latency and average latency of a histogram metric
func buildHistLatencyGraph(panelName string, metricName string) *sdk.Panel {
	graph := sdk.NewGraph(panelName)
	addGraphPanelDefaults(graph)

	graph.GraphPanel.Yaxes[0].Format = "ms"
	graph.AddTarget(&sdk.Target{
		RefID:        "A",
		LegendFormat: "90th Percentile",
		Expr:         fmt.Sprintf("histogram_quantile(0.9, sum(rate(%s_bucket[$__rate_interval])) by (le))", metricName),
	})
	graph.AddTarget(&sdk.Target{
		RefID:        "B",
		LegendFormat: "Average",
		Expr:         fmt.Sprintf("(sum(rate(%s_sum[$__rate_interval]))) / (sum(rate(%s_count[$__rate_interval])))", metricName, metricName),
	})
	return graph
}

// Graph rate of operation of a histogram metric
func buildHistRateGraph(panelName string, metricName string) *sdk.Panel {
	graph := sdk.NewGraph(panelName)
	addGraphPanelDefaults(graph)

	graph.GraphPanel.Yaxes[0].Format = "ops"
	graph.AddTarget(&sdk.Target{
		RefID:        "A",
		LegendFormat: "Per Second",
		Expr:         fmt.Sprintf("sum(rate(%s_count[$__rate_interval]))", metricName),
	})
	return graph
}

func slugName(panelName string) string {
	return strings.ReplaceAll(strings.ToLower(panelName), " ", "_")
}

func writePanel(panelName string, panel *sdk.Panel) error {
	raw, err := json.Marshal(panel)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("_data/%s.json", slugName(panelName)), raw, 0644)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	histVecLatencyPanels := map[string]string{
		"sp_connect_dynamo_connector_group_latency_ms": "Connector Group Repo Op Latency",
		"sp_connect_dynamo_connector_latency_ms":       "Connector Repo Op Latency",
		"sp_connect_dynamo_invocation_latency_ms":      "Invocation Repo Op Latency",
		"sp_connect_queue_latency_ms":                  "Queue Op Latency",
		"sp_connect_redis_lock_latency_ms":             "Redis Lock Op Latency",
		"sp_connect_redis_kvs_latency_ms":              "Redis KVS Op Latency",
		"sp_connect_response_gateway_latency_recv_ms":  "Gateway to Connect Latency",
	}
	for metricName, panelName := range histVecLatencyPanels {
		panel := buildHistVecLatencyGraph(panelName, metricName)
		err := writePanel(panelName, panel)
		if err != nil {
			log.Fatal(err)
		}
	}

	histVecRatePanels := map[string]string{
		"sp_connect_dynamo_connector_group_latency_ms": "Connector Group Op Rate",
		"sp_connect_dynamo_connector_latency_ms":       "Connector Repo Op Rate",
		"sp_connect_dynamo_invocation_latency_ms":      "Invocation Repo Op Rate",
		"sp_connect_queue_latency_ms":                  "Queue Op Rate",
		"sp_connect_redis_lock_latency_ms":             "Redis Lock Op Rate",
		"sp_connect_redis_kvs_latency_ms":              "Redis KVS Op Rate",
	}
	for metricName, panelName := range histVecRatePanels {
		panel := buildHistVecRateGraph(panelName, metricName)
		err := writePanel(panelName, panel)
		if err != nil {
			log.Fatal(err)
		}
	}

	histLatencyPanels := map[string]string{
		"sp_connect_publish_response_kafka_latency_ms": "Publish to Kafka Latency",
		"sp_connect_publish_sqs_response_latency_ms":   "Publish to SQS Latency",
	}
	for metricName, panelName := range histLatencyPanels {
		panel := buildHistLatencyGraph(panelName, metricName)
		err := writePanel(panelName, panel)
		if err != nil {
			log.Fatal(err)
		}
	}

	histRatePanels := map[string]string{
		"sp_connect_publish_response_kafka_latency_ms": "Publish to Kafka Rate",
		"sp_connect_publish_sqs_response_latency_ms":   "Publish to SQS Rate",
	}
	for metricName, panelName := range histRatePanels {
		panel := buildHistRateGraph(panelName, metricName)
		err := writePanel(panelName, panel)
		if err != nil {
			log.Fatal(err)
		}
	}
}
