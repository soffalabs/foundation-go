package db

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"time"
)

type TimeSerieEntry struct {
	Signal    string
	Tags      map[string]string
	Fields    map[string]interface{}
	Timestamp time.Time
}

type TimeSeries interface {
	Save(entries []TimeSerieEntry)
}

type InfluxDBClient struct {
	TimeSeries
	Url    string
	Token  string
	Bucket string
	Org    string
}

func NewInfluxDBClient(url string, token string, org string, bucket string) TimeSeries {
	return &InfluxDBClient{
		Url:    url,
		Token:  token,
		Bucket: bucket,
		Org:    org,
	}
}

func (c *InfluxDBClient) Save(entries []TimeSerieEntry) {
	client := influxdb2.NewClient(c.Url, c.Token)
	defer client.Close()
	writeAPI := client.WriteAPI(c.Org, c.Bucket)
	for _, entry := range entries {
		p := influxdb2.NewPoint(entry.Signal,
			entry.Tags,
			entry.Fields,
			entry.Timestamp)
		writeAPI.WritePoint(p)
	}
	writeAPI.Flush()
}
