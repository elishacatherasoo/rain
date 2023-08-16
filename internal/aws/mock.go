//go:build func_test

// Package aws contains functionality that wraps the AWS SDK
package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elishacatherasoo/rain/internal/config"
	"github.com/elishacatherasoo/rain/internal/console/spinner"
)

var awsCfg *aws.Config

var defaultSessionName = fmt.Sprintf("%s-%s", config.NAME, config.VERSION)
var lastSessionName = defaultSessionName

func loadConfig(ctx context.Context, sessionName string) *aws.Config {
	cfg := aws.Config{}

	if config.Region != "" {
		cfg.Region = config.Region
	} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
		cfg.Region = r
	} else {
		cfg.Region = "us-east-1"
	}

	lastSessionName = sessionName

	return &cfg
}

// Config loads an aws.Config based on current settings
func Config() aws.Config {
	return NamedConfig(defaultSessionName)
}

// NamedConfig loads an aws.Config based on current settings
// with configurable session name
func NamedConfig(sessionName string) aws.Config {
	message := "Loading AWS config"

	if sessionName != lastSessionName {
		message = "Reloading AWS credentials"
		awsCfg = nil
	}

	if awsCfg == nil {
		spinner.Push(message)
		awsCfg = loadConfig(context.Background(), sessionName)
		spinner.Pop()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}
