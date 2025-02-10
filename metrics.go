package metric

import (
	"context"
	"time"
)

type Type int

const (
	INT64_COUNTER Type = iota
	INT64_GAUGE
)

func (t Type) String() string {
	return [...]string{"INT64_COUNTER", "INT64_GAUGE"}[t]
}

type Metric interface {
	GetValue() int64
	GetType() Type
	GetName() *string
	GetDescription() *string
	GetUnit() *string
	GetUpdateTime() time.Time
	GetAttributes() map[string]string
}

type Int64Counter interface {
	Metric
	Add(ctx context.Context, incr int64, options ...AddOption)
}

type Int64Gauge interface {
	Metric
	Record(ctx context.Context, value int64, options ...RecordOption)
}

type AddOption interface {
}

type RecordOption interface {
}

type metric_common struct {
	value       int64
	lastUpdate  time.Time
	name        *string
	description *string
	meter       *meter
	unit        *string
	attributes  map[string]string
}

type int64Counter struct {
	metric_common
}

type int64Gauge struct {
	metric_common
}

func (c *int64Counter) Add(ctx context.Context, incr int64, options ...AddOption) {
	c.meter.addC <- &add_op{incr, c, time.Now()}
}

func (c *int64Counter) GetType() Type {
	return INT64_COUNTER
}

func (m *metric_common) GetValue() int64 {
	return m.value
}

func (m *metric_common) GetName() *string {
	return m.name
}

func (m *metric_common) GetDescription() *string {
	return m.description
}

func (m *metric_common) GetUnit() *string {
	if m.unit == nil {
		defaultUnit := "1"
		return &defaultUnit
	}
	return m.unit
}

func (m *metric_common) GetUpdateTime() time.Time {
	return m.lastUpdate
}

func (m *metric_common) GetAttributes() map[string]string {
	return m.attributes
}

func (g *int64Gauge) Record(ctx context.Context, value int64, options ...RecordOption) {
	g.meter.recC <- &rec_op{value, g, time.Now()}
}

func (g *int64Gauge) GetType() Type {
	return INT64_GAUGE
}
