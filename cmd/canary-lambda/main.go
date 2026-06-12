// Package main implements the config-drift canary Lambda (WS-F-04).
//
// It is triggered on an hourly schedule via EventBridge and performs two
// checks:
//
//  1. Is the Bedrock supervisor-agent-id still a PLACEHOLDER?
//  2. Are the Connect phone numbers still PENDING / PLACEHOLDER / empty?
//
// Results are published as CloudWatch custom metrics in namespace
// "HeadsetAgent/Canary" so that CloudWatch alarms can page on-call if
// configuration drift persists.
//
// Metric design
// -------------
//
//   - AgentIdIsPlaceholder  (no dimensions)
//     Value 1 → supervisor-agent-id is empty or "PLACEHOLDER"
//     Value 0 → param contains a real agent ID
//
//   - PhoneNumberPending  dimension Path=lex
//     Value 1 → phone-number-lex is empty, "PENDING", or "PLACEHOLDER"
//     Value 0 → param contains a real phone number
//
//   - PhoneNumberPending  dimension Path=nova
//     Value 1 → phone-number-nova-sonic is empty, "PENDING", or "PLACEHOLDER"
//     Value 0 → param contains a real phone number
//
// Two datapoints for PhoneNumberPending (one per path/dimension) give
// operators visibility into which specific path is still unconfigured.
// Alarms on the PhoneNumberPending metric can use the aggregate (no-dimension
// query) or per-path queries; the alarms in template.yaml use the aggregate.
//
// Privacy
// -------
// Raw phone number values are NEVER logged.  Only the boolean state
// (is_pending: true/false) is recorded.
package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/headset-support-agent/internal/logging"
)

const (
	cwNamespace = "HeadsetAgent/Canary"
)

// sentinel values that indicate the parameter has not been populated yet.
var sentinelValues = map[string]bool{
	"":            true,
	"PLACEHOLDER": true,
	"PENDING":     true,
}

// isPlaceholder returns true when v is empty or a known sentinel string.
// Comparison is case-insensitive to guard against accidental casing variants.
func isPlaceholder(v string) bool {
	return sentinelValues[strings.ToUpper(strings.TrimSpace(v))]
}

// handler is the Lambda entry point.  It reads three SSM parameters, derives
// boolean health states, and publishes CloudWatch custom metrics.
func handler(ctx context.Context) error {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		slog.Error("failed to load AWS config", slog.String("error", err.Error()))
		return err
	}

	ssmClient := ssm.NewFromConfig(cfg)
	cwClient := cloudwatch.NewFromConfig(cfg)

	// --- read param paths from env vars ---
	agentIDParam := mustEnv("SUPERVISOR_AGENT_ID_PARAM")
	phoneLexParam := mustEnv("PHONE_LEX_PARAM")
	phoneNovaParam := mustEnv("PHONE_NOVA_PARAM")

	// --- fetch SSM values ---
	agentIDVal, err := getParam(ctx, ssmClient, agentIDParam)
	if err != nil {
		slog.Error("failed to read supervisor-agent-id param",
			slog.String("param", agentIDParam),
			slog.String("error", err.Error()),
		)
		return err
	}

	phoneLexVal, err := getParam(ctx, ssmClient, phoneLexParam)
	if err != nil {
		slog.Error("failed to read phone-number-lex param",
			slog.String("param", phoneLexParam),
			slog.String("error", err.Error()),
		)
		return err
	}

	phoneNovaVal, err := getParam(ctx, ssmClient, phoneNovaParam)
	if err != nil {
		slog.Error("failed to read phone-number-nova-sonic param",
			slog.String("param", phoneNovaParam),
			slog.String("error", err.Error()),
		)
		return err
	}

	// --- derive boolean states (never log raw values) ---
	agentPending := isPlaceholder(agentIDVal)
	lexPending := isPlaceholder(phoneLexVal)
	novaPending := isPlaceholder(phoneNovaVal)

	slog.Info("canary check complete",
		slog.Bool("agent_id_is_placeholder", agentPending),
		slog.Bool("phone_lex_is_pending", lexPending),
		slog.Bool("phone_nova_is_pending", novaPending),
	)

	// --- build metric data ---
	metricData := []cwtypes.MetricDatum{
		// AgentIdIsPlaceholder — no dimensions; straightforward 0/1 gauge.
		{
			MetricName: aws.String("AgentIdIsPlaceholder"),
			Value:      aws.Float64(boolToFloat(agentPending)),
			Unit:       cwtypes.StandardUnitCount,
		},
		// PhoneNumberPending Path=lex
		{
			MetricName: aws.String("PhoneNumberPending"),
			Dimensions: []cwtypes.Dimension{
				{Name: aws.String("Path"), Value: aws.String("lex")},
			},
			Value: aws.Float64(boolToFloat(lexPending)),
			Unit:  cwtypes.StandardUnitCount,
		},
		// PhoneNumberPending Path=nova
		{
			MetricName: aws.String("PhoneNumberPending"),
			Dimensions: []cwtypes.Dimension{
				{Name: aws.String("Path"), Value: aws.String("nova")},
			},
			Value: aws.Float64(boolToFloat(novaPending)),
			Unit:  cwtypes.StandardUnitCount,
		},
	}

	_, err = cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(cwNamespace),
		MetricData: metricData,
	})
	if err != nil {
		slog.Error("failed to publish CloudWatch metrics",
			slog.String("namespace", cwNamespace),
			slog.String("error", err.Error()),
		)
		return err
	}

	slog.Info("metrics published",
		slog.String("namespace", cwNamespace),
		slog.Int("datapoints", len(metricData)),
	)
	return nil
}

// getParam fetches a single SSM parameter value (no decryption needed for
// plain String params).
func getParam(ctx context.Context, client *ssm.Client, name string) (string, error) {
	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return "", nil
	}
	return *out.Parameter.Value, nil
}

// mustEnv returns the value of an environment variable or logs a warning and
// returns an empty string (callers treat empty path as a missing config error).
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Warn("required environment variable not set", slog.String("key", key))
	}
	return v
}

// boolToFloat converts a boolean to a CloudWatch metric value (1.0 = true / alarm, 0.0 = ok).
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func init() {
	logging.Init()
}

func main() {
	lambda.Start(handler)
}
