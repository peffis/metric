package metric

import (
	"context"
	"fmt"
	"os"
	"time"

	otlpcollector "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const DEFAULT_EXPORT_INTERVAL = 60 * time.Second
const DEFAULT_ENDPOINT = "localhost:4317"
const DEFAULT_MAX_METRICS_EXPORT = 500

type MeterOption interface {
}

type MeterIface interface {
	Int64Counter(name, description *string, attributes ...map[string]string) Int64Counter
	Int64Gauge(name, description *string, attributes ...map[string]string) Int64Gauge
}

type meter struct {
	addC chan *add_op
	recC chan *rec_op
	regC chan Metric
}

func Meter(name string, opts ...MeterOption) MeterIface {
	meter := &meter{
		addC: make(chan *add_op, 256),
		recC: make(chan *rec_op, 256),
		regC: make(chan Metric, 256),
	}
	go meter.run()
	return meter
}

func (m *meter) Int64Counter(name, description *string, attributes ...map[string]string) Int64Counter {
	var attrs map[string]string
	if len(attributes) > 0 {
		attrs = attributes[0]
	}

	c := &int64Counter{
		metric_common: metric_common{
			name:        name,
			description: description,
			attributes:  attrs,
			meter:       m,
		},
	}

	m.regC <- c
	return c
}

func (m *meter) Int64Gauge(name, description *string, attributes ...map[string]string) Int64Gauge {
	var attrs map[string]string
	if len(attributes) > 0 {
		attrs = attributes[0]
	}

	g := &int64Gauge{
		metric_common: metric_common{
			name:        name,
			description: description,
			attributes:  attrs,
			meter:       m,
		},
	}

	m.regC <- g
	return g
}

type add_op struct {
	incr    int64
	counter *int64Counter
	when    time.Time
}

type rec_op struct {
	value int64
	gauge *int64Gauge
	when  time.Time
}

func (m *meter) run() {
	exportInterval := DEFAULT_EXPORT_INTERVAL
	if i := os.Getenv("OTEL_METRIC_EXPORT_INTERVAL"); i != "" {
		intervalFromEnv, err := time.ParseDuration(i)
		if err == nil {
			exportInterval = intervalFromEnv
		}
	}

	exportEndpoint := DEFAULT_ENDPOINT
	if e := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"); e != "" {
		exportEndpoint = e
	}

	conn, err := grpc.NewClient(exportEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	client := otlpcollector.NewMetricsServiceClient(conn)

	ticker := time.NewTicker(exportInterval)
	defer ticker.Stop()

	var metrics []Metric
	for {
		select {
		case m := <-m.regC:
			metrics = append(metrics, m)

		case ao := <-m.addC:
			ao.counter.value += ao.incr
			ao.counter.lastUpdate = ao.when

		case ro := <-m.recC:
			ro.gauge.value = ro.value
			ro.gauge.lastUpdate = ro.when

		case <-ticker.C:
			metricsToExport := make([]*metricspb.Metric, 0, DEFAULT_MAX_METRICS_EXPORT)
			for mi, m := range metrics {
				metric := makeMetricPayload(m)
				metricsToExport = append(metricsToExport, metric)
				if len(metricsToExport) == DEFAULT_MAX_METRICS_EXPORT ||
					mi == len(metrics)-1 {
					exportMetrics(client, metricsToExport)
					metricsToExport = nil
				}
			}
			if len(metricsToExport) > 0 { // flush any left-over metrics
				exportMetrics(client, metricsToExport)
			}
		}
	}
}

func exportMetrics(client otlpcollector.MetricsServiceClient, metrics []*metricspb.Metric) error {
	request := &otlpcollector.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key:   "env",
							Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "production"}},
						},
					},
				},
				ScopeMetrics: []*metricspb.ScopeMetrics{
					{
						Scope: &commonpb.InstrumentationScope{
							Name:    "mymetric",
							Version: "1.0.0",
						},
						Metrics: metrics,
					},
				},
			},
		},
	}

	// Send the request
	_, err := client.Export(context.Background(), request)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}
	return err
}

func makeMetricPayload(m Metric) *metricspb.Metric {
	dp := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(m.GetUpdateTime().UnixNano()),
		Value: &metricspb.NumberDataPoint_AsInt{
			AsInt: m.GetValue(),
		},
		Attributes: makeAttributes(m),
	}

	metric := &metricspb.Metric{
		Name:        *m.GetName(),
		Description: *m.GetDescription(),
		Unit:        *m.GetUnit(),
	}

	switch m.GetType() {
	case INT64_COUNTER:
		metric.Data = &metricspb.Metric_Sum{
			Sum: &metricspb.Sum{
				DataPoints: []*metricspb.NumberDataPoint{dp},
			},
		}

	case INT64_GAUGE:
		metric.Data = &metricspb.Metric_Gauge{
			Gauge: &metricspb.Gauge{
				DataPoints: []*metricspb.NumberDataPoint{dp},
			},
		}
	}

	return metric
}

func makeAttributes(m Metric) []*commonpb.KeyValue {
	attrs := []*commonpb.KeyValue{}
	for k, v := range m.GetAttributes() {
		attrs = append(attrs, &commonpb.KeyValue{
			Key: k,
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{StringValue: v}},
		})
	}
	return attrs
}
