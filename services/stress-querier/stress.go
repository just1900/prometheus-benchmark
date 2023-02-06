package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	minTime = time.Unix(math.MinInt64/1000+62135596801, 0)
	maxTime = time.Unix(math.MaxInt64/1000-62135596801, 999999999)
)

func registerStress(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("stress", "Stress tests a remote Prometheus Query API.")
	workers := cmd.Flag("workers", "Number of go routines for stress testing.").Required().Int()
	timeout := cmd.Flag("timeout", "Timeout of each operation").Default("60s").Duration()
	lookback := cmd.Flag("query.look-back", "How much time into the past at max we should look back").Default("300h").Duration()
	url := cmd.Arg("url", "IP:PORT pair of the target to stress.").Required().String()

	// TODO(GiedriusS): send other requests like Info() as well.
	// TODO(GiedriusS): we could ask for random aggregations.
	m["stress"] = func(g *run.Group, logger log.Logger) error {
		mainCtx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			cli, err := newQueryClient(*url)
			if err != nil {
				return err
			}

			lblvlsCtx, lblvlsCancel := context.WithTimeout(mainCtx, *timeout)
			defer lblvlsCancel()

			// get all the LabelValues
			labelvalues, warnings, err := cli.LabelValues(lblvlsCtx, labels.MetricName, []string{}, minTime, maxTime)
			if err != nil {
				return err
			}
			if len(warnings) > 0 {
				return errors.New(fmt.Sprintf("got %#v warnings from LabelValues() call", warnings))
			}
			if len(labelvalues) == 0 {
				return errors.New("the StoreAPI responded with zero metric names")
			}

			errg, errCtx := errgroup.WithContext(mainCtx)

			for i := 0; i < *workers; i++ {
				errg.Go(func() error {
					for {
						select {
						case <-errCtx.Done():
							return nil
						default:
						}

						opCtx, cancel := context.WithTimeout(errCtx, *timeout)
						defer cancel()

						randomMetric := labelvalues[rand.Intn(len(labelvalues))]
						max := time.Now()
						// subtracting
						min := max.Add(-time.Duration(rand.Int63n(int64(lookback.Nanoseconds()))))

						level.Info(logger).Log("msg", "querying metric "+randomMetric, "start", min, "end", max, "duration", max.Sub(min).String())
						_, _, err = cli.QueryRange(opCtx, string(randomMetric), v1.Range{
							Start: min,
							End:   max,
							Step:  30 * time.Second,
						})

						if err != nil {
							return err
						}
					}
				})
			}

			return errg.Wait()
		}, func(err error) {
			if err != nil {
				level.Info(logger).Log("msg", "stress test encountered an error", "err", err.Error())
			}
			cancel()
		})
		return nil
	}
}

func newQueryClient(url string) (v1.API, error) {
	apiClient, err := api.NewClient(api.Config{
		Address: url,
	})

	if err != nil {
		return nil, err
	}
	return v1.NewAPI(apiClient), nil
}
