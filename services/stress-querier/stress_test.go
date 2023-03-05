package main

import (
	"context"
	"log"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func TestNewQueryClient(t *testing.T) {
	cli, _ := newQueryClient("http://localhost:9090")
	endt := time.Now()
	result, _, _ := cli.QueryRange(context.TODO(), "ALERTS_FOR_STATE", v1.Range{
		Start: time.Now().Add(-time.Hour),
		End:   endt,
		Step:  30 * time.Second,
	})
	samples := result.(model.Matrix)
	for _, sample := range samples {
		// we need to detect the timestamp change
		end := sample.Values[len(sample.Values)-1].Timestamp
		log.Println(end)
		log.Println(endt)
		log.Println(end.Time().Unix() == endt.Unix())
	}

}
