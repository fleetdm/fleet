package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// awsSources reads a load-test instance's infrastructure. The load-test terraform names the ECS cluster, the Aurora
// cluster and the Fleet log group after the instance (fleet-<workspace>), the Fleet ECS service "fleet", and the
// internal (device) load balancer <instance>-int.
type awsSources struct {
	lt   string
	ecs  *ecs.Client
	cw   *cloudwatch.Client
	logs *cloudwatchlogs.Client

	albOnce sync.Once
	alb     string // the LoadBalancer dimension, app/<name>/<id>
	albErr  error
}

const (
	cwLag   = 3 * time.Minute // ECS, RDS and ALB metrics settle a minute or two late
	logsLag = time.Minute
)

func newAWSSources(ctx context.Context, lt, region string) (*awsSources, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS config: %w", err)
	}
	return &awsSources{
		lt:   lt,
		ecs:  ecs.NewFromConfig(cfg),
		cw:   cloudwatch.NewFromConfig(cfg),
		logs: cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

func (a *awsSources) source(name string) (source, unit, error) {
	ecsDims := []cwtypes.Dimension{
		{Name: aws.String("ClusterName"), Value: aws.String(a.lt)},
		{Name: aws.String("ServiceName"), Value: aws.String("fleet")},
	}
	role := func(r string) []cwtypes.Dimension {
		return []cwtypes.Dimension{
			{Name: aws.String("DBClusterIdentifier"), Value: aws.String(a.lt)},
			{Name: aws.String("Role"), Value: aws.String(r)},
		}
	}
	switch name {
	case "fleet.memory.max":
		return &cwSource{a: a, ns: "AWS/ECS", name: "MemoryUtilization", stat: "Maximum", dims: ecsDims}, unitPercent, nil
	case "fleet.cpu.max":
		return &cwSource{a: a, ns: "AWS/ECS", name: "CPUUtilization", stat: "Maximum", dims: ecsDims}, unitPercent, nil
	case "writer.cpu":
		return &cwSource{a: a, ns: "AWS/RDS", name: "CPUUtilization", stat: "Average", dims: role("WRITER")}, unitPercent, nil
	case "reader.cpu":
		return &cwSource{a: a, ns: "AWS/RDS", name: "CPUUtilization", stat: "Average", dims: role("READER")}, unitPercent, nil
	case "fleet.tasks_missing":
		return &tasksMissingSource{a: a}, unitCount, nil
	case "fleet.task_stops":
		return &taskStopsSource{a: a}, unitCount, nil
	case "alb.5xx_pct":
		return &alb5xxSource{a: a}, unitPercent, nil
	case "checkin.errors":
		return &checkinErrorsSource{a: a}, unitCount, nil
	}
	if uri, ok := logURIs[name]; ok {
		return &logsP99Source{a: a, uri: uri}, unitSeconds, nil
	}
	return nil, 0, fmt.Errorf("unknown metric %q", name)
}

// cwSource is a CloudWatch metric at one-minute resolution: the highest datapoint in the window for Maximum, the
// mean of the datapoints for Average.
type cwSource struct {
	a    *awsSources
	ns   string
	name string
	stat string
	dims []cwtypes.Dimension
}

func (s *cwSource) lag() time.Duration { return cwLag }

func (s *cwSource) read(ctx context.Context, from, to time.Time) (reading, error) {
	vals, err := s.a.metric(ctx, s.ns, s.name, s.stat, s.dims, from, to)
	if err != nil {
		return reading{}, err
	}
	if len(vals) == 0 {
		return reading{none: true}, nil
	}
	if s.stat == "Maximum" {
		return reading{value: slices.Max(vals)}, nil
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return reading{value: sum / float64(len(vals))}, nil
}

func (a *awsSources) metric(ctx context.Context, ns, name, stat string, dims []cwtypes.Dimension, from, to time.Time) ([]float64, error) {
	out, err := a.cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(from),
		EndTime:   aws.Time(to),
		MetricDataQueries: []cwtypes.MetricDataQuery{{
			Id: aws.String("m"),
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{Namespace: aws.String(ns), MetricName: aws.String(name), Dimensions: dims},
				Period: aws.Int32(60),
				Stat:   aws.String(stat),
			},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", ns, name, err)
	}
	if len(out.MetricDataResults) == 0 {
		return nil, nil
	}
	return out.MetricDataResults[0].Values, nil
}

// tasksMissingSource is the Fleet service's desired task count minus its running count, now.
type tasksMissingSource struct{ a *awsSources }

func (s *tasksMissingSource) lag() time.Duration { return 0 }

func (s *tasksMissingSource) read(ctx context.Context, _, _ time.Time) (reading, error) {
	out, err := s.a.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{Cluster: aws.String(s.a.lt), Services: []string{"fleet"}})
	if err != nil {
		return reading{}, fmt.Errorf("describe the fleet service: %w", err)
	}
	if len(out.Services) == 0 {
		return reading{}, errors.New("no fleet service in the cluster")
	}
	svc := out.Services[0]
	return reading{
		value:  float64(max(0, svc.DesiredCount-svc.RunningCount)),
		detail: map[string]any{"running": svc.RunningCount, "desired": svc.DesiredCount},
	}, nil
}

// taskStopsSource counts Fleet tasks that stopped in the window, with their reasons (OOM, failed health checks, a
// deploy). ECS keeps stopped tasks for about an hour, so this is where those reasons survive.
type taskStopsSource struct{ a *awsSources }

func (s *taskStopsSource) lag() time.Duration { return 0 }

func (s *taskStopsSource) read(ctx context.Context, from, to time.Time) (reading, error) {
	var arns []string
	p := ecs.NewListTasksPaginator(s.a.ecs, &ecs.ListTasksInput{
		Cluster: aws.String(s.a.lt), ServiceName: aws.String("fleet"), DesiredStatus: ecstypes.DesiredStatusStopped,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return reading{}, fmt.Errorf("list stopped tasks: %w", err)
		}
		arns = append(arns, page.TaskArns...)
	}
	n := 0
	reasons := map[string]int{}
	for chunk := range slices.Chunk(arns, 100) {
		out, err := s.a.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{Cluster: aws.String(s.a.lt), Tasks: chunk})
		if err != nil {
			return reading{}, fmt.Errorf("describe stopped tasks: %w", err)
		}
		for _, t := range out.Tasks {
			if t.StoppedAt == nil || t.StoppedAt.Before(from) || !t.StoppedAt.Before(to) {
				continue
			}
			n++
			reasons[stopReason(t)]++
		}
	}
	r := reading{value: float64(n)}
	if n > 0 {
		r.detail = map[string]any{"reasons": reasons}
	}
	return r, nil
}

func stopReason(t ecstypes.Task) string {
	parts := []string{string(t.StopCode)}
	if t.StoppedReason != nil {
		parts = append(parts, *t.StoppedReason)
	}
	for _, c := range t.Containers {
		if c.Reason != nil && *c.Reason != "" {
			parts = append(parts, aws.ToString(c.Name)+": "+*c.Reason)
		}
		if c.ExitCode != nil && *c.ExitCode != 0 {
			parts = append(parts, aws.ToString(c.Name)+" exit "+strconv.Itoa(int(*c.ExitCode)))
		}
	}
	return strings.Join(parts, "; ")
}

// alb5xxSource is the device load balancer's 5xx responses (the targets' and its own 502/504s) as a percentage of
// its requests in the window.
type alb5xxSource struct{ a *awsSources }

func (s *alb5xxSource) lag() time.Duration { return cwLag }

func (s *alb5xxSource) read(ctx context.Context, from, to time.Time) (reading, error) {
	lb, err := s.a.deviceALB(ctx)
	if err != nil {
		return reading{}, err
	}
	dims := []cwtypes.Dimension{{Name: aws.String("LoadBalancer"), Value: aws.String(lb)}}
	sum := func(name string) (float64, error) {
		vals, err := s.a.metric(ctx, "AWS/ApplicationELB", name, "Sum", dims, from, to)
		total := 0.0
		for _, v := range vals {
			total += v
		}
		return total, err
	}
	req, err := sum("RequestCount")
	if err != nil {
		return reading{}, err
	}
	if req == 0 {
		return reading{none: true}, nil
	}
	target, err := sum("HTTPCode_Target_5XX_Count")
	if err != nil {
		return reading{}, err
	}
	elb, err := sum("HTTPCode_ELB_5XX_Count")
	if err != nil {
		return reading{}, err
	}
	return reading{
		value:  100 * (target + elb) / req,
		detail: map[string]any{"requests": req, "target_5xx": target, "elb_5xx": elb},
	}, nil
}

// deviceALB finds the internal load balancer's CloudWatch dimension from the metrics it publishes, so the driver
// needs no load-balancer permissions.
func (a *awsSources) deviceALB(ctx context.Context) (string, error) {
	a.albOnce.Do(func() {
		want := "app/" + a.lt + "-int/"
		p := cloudwatch.NewListMetricsPaginator(a.cw, &cloudwatch.ListMetricsInput{
			Namespace: aws.String("AWS/ApplicationELB"), MetricName: aws.String("RequestCount"),
		})
		for p.HasMorePages() {
			page, err := p.NextPage(ctx)
			if err != nil {
				a.albErr = fmt.Errorf("find the device load balancer: %w", err)
				return
			}
			for _, m := range page.Metrics {
				for _, d := range m.Dimensions {
					if aws.ToString(d.Name) == "LoadBalancer" && strings.HasPrefix(aws.ToString(d.Value), want) {
						a.alb = aws.ToString(d.Value)
						return
					}
				}
			}
		}
		a.albErr = fmt.Errorf("no load balancer metrics for %s*", want)
	})
	return a.alb, a.albErr
}

// tookSeconds turns Fleet's request log `took` (a Go duration: 850µs, 12.5ms, 1.39s, 1m2.5s) into seconds.
const tookSeconds = `parse took /^(?<v>[0-9.]+)(?<u>µs|ms|s)$/ ` +
	`| parse took /^(?<mins>[0-9]+)m(?<secs>[0-9.]+)s$/ ` +
	`| fields if(u = "µs", v / 1000000, if(u = "ms", v / 1000, if(u = "s", v, mins * 60 + secs))) as t`

// logsP99Source is one request path's server-side p99 from Fleet's request log.
type logsP99Source struct {
	a   *awsSources
	uri string
}

func (s *logsP99Source) lag() time.Duration { return logsLag }

func (s *logsP99Source) read(ctx context.Context, from, to time.Time) (reading, error) {
	q := fmt.Sprintf(`filter uri = %q | %s | stats pct(t, 99) as p99, count(t) as n, max(t) as mx`, s.uri, tookSeconds)
	rows, err := s.a.insights(ctx, q, from, to)
	if err != nil {
		return reading{}, err
	}
	if len(rows) == 0 || rows[0]["n"] == "" || rows[0]["n"] == "0" {
		return reading{none: true}, nil
	}
	p99, err := strconv.ParseFloat(rows[0]["p99"], 64)
	if err != nil {
		return reading{}, fmt.Errorf("p99 %q: %w", rows[0]["p99"], err)
	}
	n, _ := strconv.Atoi(rows[0]["n"])
	mx, _ := strconv.ParseFloat(rows[0]["mx"], 64)
	return reading{value: p99, detail: map[string]any{"requests": n, "max_s": round3(mx)}}, nil
}

// checkinErrorsSource is the busiest minute's count of "context canceled" errors in the window: requests the
// server gave up on, mostly host check-ins that waited too long for the database.
type checkinErrorsSource struct{ a *awsSources }

func (s *checkinErrorsSource) lag() time.Duration { return logsLag }

func (s *checkinErrorsSource) read(ctx context.Context, from, to time.Time) (reading, error) {
	rows, err := s.a.insights(ctx, `filter err like /context canceled/ | stats count() as n by bin(1m)`, from, to)
	if err != nil {
		return reading{}, err
	}
	peak, total := 0, 0
	for _, r := range rows {
		n, _ := strconv.Atoi(r["n"])
		peak, total = max(peak, n), total+n
	}
	return reading{value: float64(peak), detail: map[string]any{"total": total}}, nil
}

// insights runs a Logs Insights query on the Fleet log group and returns its rows as field maps.
func (a *awsSources) insights(ctx context.Context, query string, from, to time.Time) ([]map[string]string, error) {
	start, err := a.logs.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		LogGroupName: aws.String(a.lt),
		StartTime:    aws.Int64(from.Unix()),
		EndTime:      aws.Int64(to.Unix()),
		QueryString:  aws.String(query),
		Limit:        aws.Int32(1000),
	})
	if err != nil {
		return nil, fmt.Errorf("start a logs query: %w", err)
	}
	qctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		out, err := a.logs.GetQueryResults(qctx, &cloudwatchlogs.GetQueryResultsInput{QueryId: start.QueryId})
		if err != nil {
			return nil, fmt.Errorf("logs query results: %w", err)
		}
		switch out.Status {
		case logtypes.QueryStatusComplete:
			rows := make([]map[string]string, 0, len(out.Results))
			for _, r := range out.Results {
				row := make(map[string]string, len(r))
				for _, f := range r {
					row[aws.ToString(f.Field)] = aws.ToString(f.Value)
				}
				rows = append(rows, row)
			}
			return rows, nil
		case logtypes.QueryStatusFailed, logtypes.QueryStatusCancelled, logtypes.QueryStatusTimeout:
			return nil, fmt.Errorf("logs query %s", out.Status)
		}
		select {
		case <-qctx.Done():
			_, _ = a.logs.StopQuery(context.WithoutCancel(ctx), &cloudwatchlogs.StopQueryInput{QueryId: start.QueryId})
			return nil, fmt.Errorf("logs query: %w", qctx.Err())
		case <-time.After(time.Second):
		}
	}
}
