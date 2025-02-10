# metrics - a small memory footprint OTLP/GRPC metrics instrumentation library

This is just a quick experiment. It is incomplete (only supports
Int64Counter and Int64Gauge) and probably has many issues.

You can control it with two environment variables:
| Environment variable | Description | Example |
| -------------------- | ----------- | ------- |
| OTEL_EXPORTER_OTLP_METRICS_ENDPOINT | The endpoint to send the OTLP
metrics data to | OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=127.0.0.1:4317 |
| OTEL_METRIC_EXPORT_INTERVAL | How often the metric data should be
sent to the endpoint | OTEL_METRIC_EXPORT_INTERVAL=15s |

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