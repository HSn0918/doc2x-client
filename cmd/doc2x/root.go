package main

import (
	"time"

	"github.com/spf13/cobra"

	client "github.com/hsn0918/doc2x-client"
)

type cliOptions struct {
	apiKey            string
	baseURL           string
	timeout           time.Duration
	processingTimeout time.Duration
	failLogPath       string
}

func newRootCmd() *cobra.Command {
	opts := &cliOptions{}

	cmd := &cobra.Command{
		Use:           "doc2x",
		Short:         "Doc2X API v2 CLI helper",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&opts.apiKey, "api-key", "", "Doc2X API key (or set DOC2X_APIKEY / DOC2X_API_KEY)")
	cmd.PersistentFlags().StringVar(&opts.baseURL, "base-url", client.DefaultBaseURL, "Base URL for Doc2X API")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", client.DefaultTimeout, "HTTP timeout for API requests")
	cmd.PersistentFlags().DurationVar(&opts.processingTimeout, "processing-timeout", client.ProcessingTimeout, "Timeout for long running operations")
	cmd.PersistentFlags().StringVar(&opts.failLogPath, "fail-log", "fail.log", "Path to write failed task logs")

	cmd.AddCommand(newParseCmd(opts))
	cmd.AddCommand(newConvertCmd(opts))
	cmd.AddCommand(newCompletionCmd())

	return cmd
}
