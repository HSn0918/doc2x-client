package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	client "github.com/hsn0918/doc2x-client"
)

func newConvertCmd(opts *cliOptions) *cobra.Command {
	co := &convertOptions{
		opts: opts,
	}

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Trigger conversion for a parsed document",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := co.Complete(); err != nil {
				if logErr := logFailure(co.opts.failLogPath, "", co.uid, err); logErr != nil {
					return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
				}
				return err
			}

			if err := co.Validate(); err != nil {
				return err
			}

			return co.Run(cmd)
		},
	}

	co.addFlags(cmd)

	return cmd
}

type convertOptions struct {
	uid            string
	to             string
	formulaMode    string
	filename       string
	mergeCrossPage bool
	wait           bool
	interval       time.Duration
	download       bool
	output         string
	opts           *cliOptions
	targetFormat   client.ConvertFormat
	targetFormula  client.FormulaMode
	apiKey         string
}

func (o *convertOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.uid, "uid", "", "UID of the parsed document")
	cmd.Flags().StringVar(&o.to, "to", string(client.FormatMarkdown), "Target format: md|tex|docx|md_dollar")
	cmd.Flags().StringVar(&o.formulaMode, "formula-mode", string(client.FormulaModeNormal), "Formula mode: normal|dollar")
	cmd.Flags().StringVar(&o.filename, "filename", "", "Optional output filename for md/tex without extension")
	cmd.Flags().BoolVar(&o.mergeCrossPage, "merge-cross-page-forms", false, "Merge cross page tables")
	cmd.Flags().BoolVar(&o.wait, "wait", true, "Wait for conversion to finish")
	cmd.Flags().DurationVar(&o.interval, "interval", 3*time.Second, "Polling interval for conversion status")
	cmd.Flags().BoolVar(&o.download, "download", false, "Download the converted file when ready")
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Download path (used when --download is set)")
}

func (o *convertOptions) Complete() error {
	format, err := parseConvertFormat(o.to)
	if err != nil {
		return err
	}
	o.targetFormat = format

	mode, err := parseFormulaMode(o.formulaMode)
	if err != nil {
		return err
	}
	o.targetFormula = mode

	if o.interval <= 0 {
		o.interval = 3 * time.Second
	}

	return nil
}

func (o *convertOptions) Validate() error {
	if o.uid == "" {
		return errors.New("flag --uid is required")
	}
	return nil
}

func (o *convertOptions) Run(cmd *cobra.Command) error {
	apiKey, err := resolveAPIKey(o.opts)
	if err != nil {
		if logErr := logFailure(o.opts.failLogPath, "", o.uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}
	o.apiKey = apiKey

	cli := buildClient(o.apiKey, o.opts)
	ctx := cmd.Context()

	req := client.ConvertRequest{
		UID:                 o.uid,
		To:                  o.targetFormat,
		FormulaMode:         o.targetFormula,
		Filename:            o.filename,
		MergeCrossPageForms: o.mergeCrossPage,
	}

	resp, err := cli.ConvertParse(ctx, req)
	if err != nil {
		if logErr := logFailure(o.opts.failLogPath, "", o.uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	if err := printWithTrace(cmd, slog.LevelInfo, resp.TraceID, "Convert requested",
		slog.String("uid", o.uid),
		slog.String("status", string(resp.Data.Status)),
	); err != nil {
		return err
	}

	if !o.wait {
		return nil
	}

	result, err := cli.WaitForConversion(ctx, o.uid, o.interval)
	if err != nil {
		if logErr := logFailure(o.opts.failLogPath, resp.TraceID, o.uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	if err := printWithTrace(cmd, slog.LevelInfo, result.TraceID, "Conversion finished",
		slog.String("status", string(result.Data.Status)),
		slog.String("uid", o.uid),
		slog.String("url", result.Data.URL),
	); err != nil {
		return err
	}

	if o.download {
		outPath := o.output
		if outPath == "" {
			outPath = defaultDownloadName(result.Data.URL, o.uid)
		}

		if err := downloadToFile(ctx, cli, result.Data.URL, outPath); err != nil {
			if logErr := logFailure(o.opts.failLogPath, result.TraceID, o.uid, err); logErr != nil {
				return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
			}
			return err
		}

		if err := printWithTrace(cmd, slog.LevelInfo, result.TraceID, "Downloaded converted file",
			slog.String("path", outPath),
		); err != nil {
			return err
		}
	}

	return nil
}
