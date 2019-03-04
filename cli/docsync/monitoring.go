package main

import (
	"net/http"
	"os"
	"os/user"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/api/option"
)

var (
	uploadedFilesCounter    = stats.Int64("uploaded_files", "The number of files uploaded.", "1")
	uploadedFilesErrCounter = stats.Int64("uploaded_files_errors", "The number of errors when uploading.", "1")
)

func setupPrometheusExport(mux *http.ServeMux) error {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "docsync",
	})
	if err != nil {
		return err
	}
	mux.Handle("/metrics", pe)
	view.RegisterExporter(pe)
	return nil
}

func setupStackdriverExport(projectID string, creds []byte) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	labels := &stackdriver.Labels{}
	labels.Set("hostname", hostname, "Hostname on which docsync is running.")
	labels.Set("user", user.Username, "Username running docsync.")

	sd, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID,
		MonitoringClientOptions: []option.ClientOption{
			option.WithCredentialsJSON(creds),
		},
		TraceClientOptions: []option.ClientOption{
			option.WithCredentialsJSON(creds),
		},
		DefaultMonitoringLabels: labels,
		Location:                "client",
		Timeout:                 1 * time.Minute,
		MetricPrefix:            "github.com/andreich/docsync",
	})
	if err == nil {
		view.RegisterExporter(sd)
	}
	return err
}

func registerViews() error {
	return view.Register(&view.View{
		Name:        "uploaded_files_count",
		Description: "Number of files uploaded over time",
		Measure:     uploadedFilesCounter,
		Aggregation: view.Count(),
	}, &view.View{
		Name:        "uploaded_files_error_count",
		Description: "Number of errors over time",
		Measure:     uploadedFilesErrCounter,
		Aggregation: view.Count(),
	})
}
