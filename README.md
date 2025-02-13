# mini metrics - a small memory footprint OTLP/GRPC metrics instrumentation library

This is a quick experiment to explore the possibility for an alternative to the official [golang OpenTelemetry metrics SDK](https://github.com/open-telemetry/opentelemetry-go). The reason for exploring the alternative was that [it was found](https://github.com/open-telemetry/opentelemetry-go/issues/6260) that the official library used much more memory than the equivalent Prometheus instrumentation. So the purpose was to explore if some, more minimal memory footprint, solution could be found that solved my needs. The official library is a great piece of software, capable of many things (not only metrics) and built in a nice, generic way, but perhaps the focus has, so far, not been on conserving memory and, some features, even if you don't use them, use memory and, since the memory is counted per metrics instrumentation (such as counters and gauges) this can amount to quite some numbers for services that use many metric instrumentations. 

This library, an experiment as it is, is incomplete though (it only supports
Int64Counter and Int64Gauge and the only export is OTLP/GRPC) and probably has many issues. I have used it for my experiments, where it was shown that it uses about 5 times less memory than the official library for an equivalent service, but it is not recommended for production environments. 

You can control it with two environment variables:
| Environment variable | Description | Example value | Default value |
| -------------------- | ----------- | ------- | --------------------|
| OTEL_EXPORTER_OTLP_METRICS_ENDPOINT | The endpoint to send the OTLP metrics data to | otel-collector:4317 | localhost:4317 |
| OTEL_METRIC_EXPORT_INTERVAL | How often the metric data should be sent to the endpoint | 15s | 60s |

## Example usage
```golang
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/peffis/metric"
)

const N_COUNTERS = 1000

var counters []metric.Int64Counter
var meter = metric.Meter("my_meter")

func init() {
	// setup the counters
	description := "A counter that gets updated periodically"
	for i := 0; i < N_COUNTERS; i++ {
		name := fmt.Sprintf("counter_%d", i)
		counter := meter.Int64Counter(
			&name,
			&description,
		)

		counters = append(counters, counter)
	}
}

func main() {
	// Increment the counter every second
	go func() {
		ctx := context.Background()
		for {
			for i := 0; i < N_COUNTERS; i++ {
				counters[i].Add(ctx, 1)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	ch := make(chan bool)
	<-ch
}
```
