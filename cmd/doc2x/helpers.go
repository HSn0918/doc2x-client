package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	client "github.com/hsn0918/doc2x-client"
)

func buildClient(apiKey string, opts *cliOptions) client.Client {
	options := []client.Option{
		client.WithAPIKey(apiKey),
		client.WithBaseURL(opts.baseURL),
		client.WithTimeout(opts.timeout),
		client.WithProcessingTimeout(opts.processingTimeout),
	}
	return client.NewClient(apiKey, options...)
}

func resolveAPIKey(opts *cliOptions) (string, error) {
	if opts.apiKey != "" {
		return opts.apiKey, nil
	}

	if env := os.Getenv("DOC2X_APIKEY"); env != "" {
		opts.apiKey = env
		return env, nil
	}

	if env := os.Getenv("DOC2X_API_KEY"); env != "" {
		opts.apiKey = env
		return env, nil
	}

	return "", errors.New("api key is required (flag --api-key or DOC2X_APIKEY / DOC2X_API_KEY)")
}

func parseConvertFormat(to string) (client.ConvertFormat, error) {
	switch strings.ToLower(to) {
	case string(client.FormatMarkdown):
		return client.FormatMarkdown, nil
	case string(client.FormatTex):
		return client.FormatTex, nil
	case string(client.FormatDocx):
		return client.FormatDocx, nil
	case string(client.FormatMDDollar):
		return client.FormatMDDollar, nil
	default:
		return "", fmt.Errorf("unsupported target format: %s", to)
	}
}

func parseFormulaMode(mode string) (client.FormulaMode, error) {
	switch strings.ToLower(mode) {
	case string(client.FormulaModeNormal):
		return client.FormulaModeNormal, nil
	case string(client.FormulaModeDollar):
		return client.FormulaModeDollar, nil
	case string(client.FormulaModeLatex):
		return client.FormulaModeLatex, nil
	default:
		return "", fmt.Errorf("unsupported formula mode: %s", mode)
	}
}

func writeJSON(path string, data any) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func defaultDownloadName(urlStr, uid string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return uid + ".zip"
	}

	ext := path.Ext(parsed.Path)
	if ext == "" {
		ext = ".zip"
	}

	return uid + ext
}

func downloadToFile(ctx context.Context, cli client.Client, downloadURL, targetPath string) error {
	dir := filepath.Dir(targetPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create download dir: %w", err)
		}
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := cli.DownloadFileTo(ctx, downloadURL, file); err != nil {
		return err
	}

	return nil
}

func printOut(cmd *cobra.Command, format string, args ...any) error {
	return logWith(cmd, slog.LevelInfo, "info", "", format, args...)
}

func printWithTrace(cmd *cobra.Command, level string, traceID string, format string, args ...any) error {
	lvl := slog.LevelInfo
	if strings.ToLower(level) == "error" {
		lvl = slog.LevelError
	}
	return logWith(cmd, lvl, level, traceID, format, args...)
}

func logWith(cmd *cobra.Command, level slog.Level, levelName string, traceID string, format string, args ...any) error {
	logger := newLogger(cmd.OutOrStdout(), level)
	msg := strings.TrimSuffix(fmt.Sprintf(format, args...), "\n")
	attrs := []slog.Attr{slog.Time("ts", time.Now())}
	if traceID != "" {
		attrs = append(attrs, slog.String("trace-id", traceID))
	}
	logger.LogAttrs(cmd.Context(), level, msg, attrs...)
	return nil
}

func newLogger(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	return slog.New(handler)
}
