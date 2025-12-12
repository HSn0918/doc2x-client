package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	client "github.com/hsn0918/doc2x-client"
)

func newParseCmd(opts *cliOptions) *cobra.Command {
	po := &parseOptions{
		opts: opts,
	}

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Upload and parse a PDF (single file or directory)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := po.Complete(); err != nil {
				target := po.inputPath
				if target == "" {
					target = po.filePath
				}
				if logErr := logFailure(po.opts.failLogPath, "", target, err); logErr != nil {
					return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
				}
				return err
			}

			if err := po.Validate(); err != nil {
				return err
			}

			return po.Run(cmd)
		},
	}

	po.addFlags(cmd)

	return cmd
}

type parseOptions struct {
	filePath    string
	inputPath   string
	wait        bool
	interval    time.Duration
	output      string
	outputDir   string
	concurrency int
	opts        *cliOptions
	files       []string
	apiKey      string
	auto        autoConvertConfig
}

type autoConvertConfig struct {
	enabled        bool
	to             string
	formula        string
	downloadDir    string
	filename       string
	mergeCrossPage bool
	output         string
}

type parseJobConfig struct {
	wait      bool
	interval  time.Duration
	output    string
	outputDir string
	failLog   string
	auto      autoConvertConfig
}

func (o *parseOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.filePath, "file", "f", "", "PDF file path to upload")
	cmd.Flags().StringVarP(&o.inputPath, "path", "p", "", "Path to a PDF file or a directory containing PDFs")
	cmd.Flags().BoolVar(&o.wait, "wait", true, "Wait for parsing to finish")
	cmd.Flags().DurationVar(&o.interval, "interval", 3*time.Second, "Polling interval for parsing status")
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Optional path to save parsed result JSON")
	cmd.Flags().StringVar(&o.outputDir, "output-dir", "", "Directory to store JSON results when parsing multiple files")
	cmd.Flags().IntVar(&o.concurrency, "concurrency", 3, "Number of concurrent uploads when using --path")
	cmd.Flags().BoolVar(&o.auto.enabled, "convert", true, "After parse success, trigger conversion and download")
	cmd.Flags().StringVar(&o.auto.to, "convert-to", string(client.FormatMarkdown), "Target format for auto conversion: md|tex|docx|md_dollar")
	cmd.Flags().StringVar(&o.auto.formula, "convert-formula-mode", string(client.FormulaModeNormal), "Formula mode for auto conversion: normal|dollar")
	cmd.Flags().StringVar(&o.auto.downloadDir, "download-dir", ".", "Directory to store auto-downloaded converted files")
	cmd.Flags().StringVar(&o.auto.filename, "convert-filename", "", "Optional output filename (md/tex) without extension during auto conversion")
	cmd.Flags().BoolVar(&o.auto.mergeCrossPage, "convert-merge-cross-page-forms", false, "Merge cross page tables during auto conversion")
	cmd.Flags().StringVar(&o.auto.output, "convert-output", "", "Override download path for auto conversion (defaults to UID-based name under download-dir)")
}

func (o *parseOptions) Complete() error {
	if o.filePath == "" && o.inputPath == "" {
		return errors.New("flag --file or --path is required")
	}

	if o.concurrency <= 0 {
		o.concurrency = 3
	}

	targetPath := o.filePath
	if targetPath == "" {
		targetPath = o.inputPath
	}

	files, err := collectInputFiles(targetPath)
	if err != nil {
		return err
	}
	o.files = files

	return nil
}

func (o *parseOptions) Validate() error {
	if len(o.files) == 0 {
		return fmt.Errorf("no pdf files found in %s", o.inputPath)
	}
	return nil
}

func (o *parseOptions) Run(cmd *cobra.Command) error {
	apiKey, err := resolveAPIKey(o.opts)
	if err != nil {
		if logErr := logFailure(o.opts.failLogPath, "", "", err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}
	o.apiKey = apiKey

	cli := buildClient(o.apiKey, o.opts)
	ctx := cmd.Context()

	jobCfg := parseJobConfig{
		wait:      o.wait,
		interval:  o.interval,
		output:    o.output,
		outputDir: o.outputDir,
		failLog:   o.opts.failLogPath,
		auto:      o.auto,
	}

	if len(o.files) == 1 {
		return handleParseFile(ctx, cmd, cli, o.files[0], jobCfg)
	}

	return runParseBatch(ctx, cmd, cli, o.files, o.concurrency, jobCfg)
}

func collectInputFiles(p string) ([]string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.Mode().IsRegular() {
		if strings.EqualFold(filepath.Ext(p), ".pdf") {
			return []string{p}, nil
		}
		return nil, fmt.Errorf("file is not a pdf: %s", p)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is neither file nor directory: %s", p)
	}

	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			files = append(files, filepath.Join(p, entry.Name()))
		}
	}

	return files, nil
}

func handleParseFile(ctx context.Context, cmd *cobra.Command, cli client.Client, pdf string, job parseJobConfig) error {
	fileLabel := filepath.Base(pdf)

	fileData, err := os.ReadFile(pdf)
	if err != nil {
		if logErr := logFailure(job.failLog, "", pdf, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return fmt.Errorf("read file %s: %w", pdf, err)
	}

	preUpload, err := cli.PreUpload(ctx)
	if err != nil {
		if logErr := logFailure(job.failLog, "", pdf, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return fmt.Errorf("preupload failed for %s: %w", pdf, err)
	}

	if err := printWithTrace(cmd, slog.LevelInfo, preUpload.TraceID, "Preupload completed",
		slog.String("file", fileLabel),
		slog.String("uid", preUpload.Data.UID),
	); err != nil {
		return err
	}

	if err := cli.UploadToPresignedURL(ctx, preUpload.Data.URL, fileData); err != nil {
		if logErr := logFailure(job.failLog, preUpload.TraceID, pdf, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return fmt.Errorf("[%s] upload failed (trace-id: %s): %w", fileLabel, preUpload.TraceID, err)
	}

	if err := printWithTrace(cmd, slog.LevelInfo, preUpload.TraceID, "Upload success",
		slog.String("file", fileLabel),
	); err != nil {
		return err
	}

	if !job.wait {
		return printWithTrace(cmd, slog.LevelInfo, preUpload.TraceID, "Submitted parse job",
			slog.String("file", fileLabel),
			slog.String("uid", preUpload.Data.UID),
		)
	}

	status, err := cli.WaitForParsing(ctx, preUpload.Data.UID, job.interval)
	if err != nil {
		if logErr := logFailure(job.failLog, preUpload.TraceID, pdf, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	if status.Data == nil {
		msgErr := fmt.Errorf("parse finished uid=%s but data is nil", preUpload.Data.UID)
		if logErr := logFailure(job.failLog, status.TraceID, pdf, msgErr); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", msgErr, logErr)
		}
		return printWithTrace(cmd, slog.LevelError, status.TraceID, "Parse finished without data",
			slog.String("file", fileLabel),
			slog.String("uid", preUpload.Data.UID),
		)
	}

	pageCount := 0
	if status.Data.Result != nil {
		pageCount = len(status.Data.Result.Pages)
	}

	if err := printWithTrace(cmd, slog.LevelInfo, status.TraceID, "Parse success",
		slog.String("file", fileLabel),
		slog.String("uid", preUpload.Data.UID),
		slog.Int("pages", pageCount),
	); err != nil {
		return err
	}

	target := job.output
	if job.outputDir != "" {
		target = filepath.Join(job.outputDir, changeExt(filepath.Base(pdf), ".json"))
	}

	if target != "" && status.Data.Result != nil {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			if logErr := logFailure(job.failLog, status.TraceID, pdf, err); logErr != nil {
				return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
			}
			return fmt.Errorf("create output dir: %w", err)
		}
		if err := writeJSON(target, status.Data.Result); err != nil {
			if logErr := logFailure(job.failLog, status.TraceID, pdf, err); logErr != nil {
				return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
			}
			return err
		}
		if err := printWithTrace(cmd, slog.LevelInfo, status.TraceID, "Saved parse result",
			slog.String("file", fileLabel),
			slog.String("path", target),
		); err != nil {
			return err
		}
	}

	if job.auto.enabled {
		if err := autoConvertAndDownload(ctx, cmd, cli, preUpload.Data.UID, job.auto, job.interval, job.failLog, filepath.Base(pdf)); err != nil {
			return err
		}
	}

	return nil
}

func changeExt(name, ext string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return base + ext
}

func runParseBatch(ctx context.Context, cmd *cobra.Command, cli client.Client, files []string, concurrency int, job parseJobConfig) error {
	eg, ctx := errgroup.WithContext(ctx)
	if concurrency > 0 {
		eg.SetLimit(concurrency)
	}

	var (
		errs []error
		mu   sync.Mutex
	)

	for _, pdf := range files {
		pdf := pdf
		eg.Go(func() error {
			if err := handleParseFile(ctx, cmd, cli, pdf, job); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch completed with %d errors, first: %w", len(errs), errs[0])
	}

	return nil
}

func autoConvertAndDownload(ctx context.Context, cmd *cobra.Command, cli client.Client, uid string, cfg autoConvertConfig, interval time.Duration, failLog string, label string) error {
	format, err := parseConvertFormat(cfg.to)
	if err != nil {
		if logErr := logFailure(failLog, "", uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	mode, err := parseFormulaMode(cfg.formula)
	if err != nil {
		if logErr := logFailure(failLog, "", uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	req := client.ConvertRequest{
		UID:                 uid,
		To:                  format,
		FormulaMode:         mode,
		Filename:            cfg.filename,
		MergeCrossPageForms: cfg.mergeCrossPage,
	}

	resp, err := cli.ConvertParse(ctx, req)
	if err != nil {
		if logErr := logFailure(failLog, "", uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	if err := printWithTrace(cmd, slog.LevelInfo, resp.TraceID, "Convert requested",
		slog.String("file", label),
		slog.String("uid", uid),
		slog.String("status", string(resp.Data.Status)),
	); err != nil {
		return err
	}

	result, err := cli.WaitForConversion(ctx, uid, interval)
	if err != nil {
		if logErr := logFailure(failLog, resp.TraceID, uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	if err := printWithTrace(cmd, slog.LevelInfo, result.TraceID, "Conversion finished",
		slog.String("file", label),
		slog.String("status", string(result.Data.Status)),
		slog.String("uid", uid),
		slog.String("url", result.Data.URL),
	); err != nil {
		return err
	}

	if cfg.output != "" {
		if err := downloadToFile(ctx, cli, result.Data.URL, cfg.output); err != nil {
			if logErr := logFailure(failLog, result.TraceID, uid, err); logErr != nil {
				return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
			}
			return err
		}
		return printWithTrace(cmd, slog.LevelInfo, result.TraceID, "Downloaded converted file",
			slog.String("file", label),
			slog.String("path", cfg.output),
		)
	}

	downloadDir := cfg.downloadDir
	if downloadDir == "" {
		downloadDir = "."
	}
	outPath := filepath.Join(downloadDir, defaultDownloadName(result.Data.URL, uid))

	if err := downloadToFile(ctx, cli, result.Data.URL, outPath); err != nil {
		if logErr := logFailure(failLog, result.TraceID, uid, err); logErr != nil {
			return fmt.Errorf("%w; also failed to write fail log: %v", err, logErr)
		}
		return err
	}

	return printWithTrace(cmd, slog.LevelInfo, result.TraceID, "Downloaded converted file",
		slog.String("file", label),
		slog.String("path", outPath),
	)
}
