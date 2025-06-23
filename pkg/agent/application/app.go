package application

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	gcplogs "github.com/jlewi/monogo/gcp/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

type App struct {
	Config         *config.Config
	otelShutdownFn func()
	logClosers     []logCloser
}

// NewApp creates a new application. You should call one more setup/Load functions to properly set it up.
func NewApp() *App {
	return &App{}
}

// LoadConfig loads the config. It takes an optional command. The command allows values to be overwritten from
// the CLI.
func (a *App) LoadConfig(cmd *cobra.Command) error {
	// N.B. at this point we haven't configured any logging so zap just returns the default logger.
	// TODO(jeremy): Should we just initialize the logger without cfg and then reinitialize it after we've read the config?
	if err := config.InitViper(cmd); err != nil {
		return err
	}
	cfg := config.GetConfig()
	if problems := cfg.IsValid(); len(problems) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Invalid configuration; %s\n", strings.Join(problems, "\n"))
		return fmt.Errorf("invalid configuration; fix the problems and then try again")
	}
	a.Config = cfg

	return nil
}

func (a *App) GetConfig() *config.Config {
	return a.Config
}

func (a *App) SetupLogging() error {
	// We don't log to a file for random CLI commands; e.g. setting/getting configuration
	return a.setupLoggers(false)
}

func (a *App) SetupServerLogging() error {
	// For the server we log to a file
	return a.setupLoggers(true)
}

// SetupOTEL sets up OpenTelemetry. Call this function if you want to enable OpenTelemetry.
func (a *App) SetupOTEL() error {
	log := zapr.NewLogger(zap.L())
	if a.Config == nil {
		return errors.New("config shouldn't be nil; did you forget to call LoadConfig?")
	}

	var tpOptions []trace.TracerProviderOption

	// Always set the resource
	tpOptions = append(tpOptions, trace.WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("cloud-assistant"),
	)))

	// Only set up OTLP HTTP exporter if endpoint is configured.
	if a.Config != nil && a.Config.Telemetry != nil && a.Config.Telemetry.OtlpHTTPEndpoint != "" {
		endpoint := a.Config.Telemetry.OtlpHTTPEndpoint
		log.Info("Setting up OTLP HTTP exporter", "endpoint", endpoint)
		exp, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
		if err != nil {
			return errors.Wrap(err, "failed to create OTLP HTTP exporter")
		}
		tpOptions = append(tpOptions, trace.WithBatcher(exp))
	} else {
		log.Info("OTLP HTTP endpoint not specified; skipping OTLP exporter setup")
	}

	tracerProvider := trace.NewTracerProvider(tpOptions...)
	otel.SetTracerProvider(tracerProvider)
	a.otelShutdownFn = func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log := zapr.NewLogger(zap.L())
			log.Error(err, "Error shutting down tracer provider")
		}
	}

	// Set otelhttp.DefaultClient to use a transport that will report metrics.
	// For other http.Clients, wrap the Transport with otelhttp.NewTransport
	// to enable OpenTelemetry instrumentation.
	otelhttp.DefaultClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	return nil
}

func (a *App) setupLoggers(logToFile bool) error {
	if a.Config == nil {
		return errors.New("Config is nil; call LoadConfig first")
	}

	cores := make([]zapcore.Core, 0, 2)

	consolePaths := make([]string, 0, 1)
	jsonPaths := make([]string, 0, 1)

	if logToFile {
		for _, sink := range a.Config.Logging.Sinks {
			project, logName, isLog := gcplogs.ParseURI(sink.Path)
			if isLog {
				if err := gcplogs.RegisterSink(project, logName, nil); err != nil {
					return err
				}
			}
			if sink.JSON {
				jsonPaths = append(jsonPaths, sink.Path)
			} else {
				consolePaths = append(consolePaths, sink.Path)
			}
		}

		// We always write raw logs in JSON format to the rawLogFile because they are used for AI traces and learning.
		// TODO(jeremy): If we are using Google Cloud Logging it would be nice if learning could just use that.
		rawLogFile, err := a.getRawLogFile()
		if err != nil {
			return errors.Wrap(err, "Could not create raw log file")
		}
		jsonPaths = append(jsonPaths, rawLogFile)
	}

	if len(consolePaths) == 0 && len(jsonPaths) == 0 {
		// If no sinks are specified we default to console logging.
		consolePaths = []string{"stderr"}
	}

	if len(consolePaths) > 0 {
		consoleCore, err := a.createCoreForConsole(consolePaths)
		if err != nil {
			return errors.Wrap(err, "Could not create core logger for console")
		}
		cores = append(cores, consoleCore)
	}

	if len(jsonPaths) > 0 {
		jsonCore, err := a.createJSONCoreLogger(jsonPaths)
		if err != nil {
			return errors.Wrap(err, "Could not create core logger for JSON paths")
		}
		cores = append(cores, jsonCore)
	}

	// Create a multi-core logger with different encodings
	core := zapcore.NewTee(cores...)

	// Create the logger
	newLogger := zap.New(core)
	// Record the caller of the log message
	newLogger = newLogger.WithOptions(zap.AddCaller())
	if a.Config.Metadata.Name != "" {
		newLogger = newLogger.With(zap.String("agentName", a.Config.Metadata.Name))
	}
	zap.ReplaceGlobals(newLogger)
	return nil
}

func (a *App) createCoreForConsole(paths []string) (zapcore.Core, error) {
	// Configure encoder for non-JSON format (console-friendly)
	c := zap.NewDevelopmentEncoderConfig()

	// Use the keys used by cloud logging
	// https://cloud.google.com/logging/docs/structured-logging
	logFields := a.Config.Logging.LogFields
	if logFields == nil {
		logFields = &config.LogFields{}
	}
	if logFields.Level != "" {
		c.LevelKey = logFields.Level
	} else {
		c.LevelKey = "severity"
	}

	if logFields.Time != "" {
		c.TimeKey = logFields.Time
	} else {
		c.TimeKey = "time"
	}

	if logFields.Message != "" {
		c.MessageKey = logFields.Message
	} else {
		c.MessageKey = "message"
	}

	lvl := a.Config.GetLogLevel()
	zapLvl := zap.NewAtomicLevel()

	if err := zapLvl.UnmarshalText([]byte(lvl)); err != nil {
		return nil, errors.Wrapf(err, "Could not convert level %v to ZapLevel", lvl)
	}

	oFile, closer, err := zap.Open(paths...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create writer for paths %s", paths)
	}
	if a.logClosers == nil {
		a.logClosers = []logCloser{}
	}
	a.logClosers = append(a.logClosers, closer)

	encoder := zapcore.NewConsoleEncoder(c)
	core := zapcore.NewCore(encoder, oFile, zapLvl)
	return core, nil
}

// getRawLogFile gets the file to write raw logs to. The file is written in JSON format.
// Ensures the directory exists.
func (a *App) getRawLogFile() (string, error) {
	logDir := filepath.Join(a.Config.GetLogDir(), "raw")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		// Logger won't be setup yet so we can't use it.
		_, _ = fmt.Fprintf(os.Stdout, "Creating log directory %s\n", logDir)
		err := os.MkdirAll(logDir, 0o755)
		if err != nil {
			return "", errors.Wrapf(err, "could not create log directory %s", logDir)
		}
	}

	// We need to set a unique file name for the logs as a way of dealing with log rotation.
	name := fmt.Sprintf("logs.%s.json", time.Now().Format("2006-01-02T15:04:05"))
	logFile := filepath.Join(logDir, name)

	return logFile, nil
}

// createJSONCoreLogger creates a core logger that writes logs in JSON format. These include raw logs which
// always written in JSON  format. Their purpose is to capture AI traces that we use for retraining.
// Since these are supposed to be machine  readable they are always written in JSON format.
func (a *App) createJSONCoreLogger(paths []string) (zapcore.Core, error) {
	// Configure encoder for JSON format
	c := zap.NewProductionEncoderConfig()
	// Use the keys used by cloud logging
	// https://cloud.google.com/logging/docs/structured-logging
	logFields := a.Config.Logging.LogFields
	if logFields == nil {
		logFields = &config.LogFields{}
	}
	if logFields.Level != "" {
		c.LevelKey = logFields.Level
	} else {
		c.LevelKey = "severity"
	}

	if logFields.Time != "" {
		c.TimeKey = logFields.Time
	} else {
		c.TimeKey = "time"
	}

	if logFields.Message != "" {
		c.MessageKey = logFields.Message
	} else {
		c.MessageKey = "message"
	}

	// We attach the function key to the logs because that is useful for identifying the function that generated the log.
	// N.B are logs processing depends on this field being present in the logs. This is one reason
	// why we don't allow it to be customized to match the field expected by a logging backend like Datadog
	// or Cloud Logging
	c.FunctionKey = "function"

	jsonEncoder := zapcore.NewJSONEncoder(c)

	_, _ = fmt.Fprintf(os.Stdout, "Writing JSON logs to %s\n", paths)

	oFile, closer, err := zap.Open(paths...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open paths %s", paths)
	}
	if a.logClosers == nil {
		a.logClosers = []logCloser{}
	}
	a.logClosers = append(a.logClosers, closer)

	zapLvl := zap.NewAtomicLevel()

	if err := zapLvl.UnmarshalText([]byte(a.Config.GetLogLevel())); err != nil {
		return nil, errors.Wrapf(err, "Could not convert level %v to ZapLevel", a.Config.GetLogLevel())
	}

	// Force log level to be at least info. Because info is the level at which we capture the logs we need for
	// tracing.
	if zapLvl.Level() > zapcore.InfoLevel {
		zapLvl.SetLevel(zapcore.InfoLevel)
	}

	core := zapcore.NewCore(jsonEncoder, zapcore.AddSync(oFile), zapLvl)

	return core, nil
}

func (a *App) Shutdown() error {
	l := zap.L()
	log := zapr.NewLogger(l)

	if a.otelShutdownFn != nil {
		log.Info("Shutting down open telemetry")
		a.otelShutdownFn()
	}

	log.Info("Shutting down the application")
	// Flush the logs
	// We do a log sync here. To try to flush any logs that are buffered.
	// Per https://github.com/jlewi/foyle/issues/295 it looks for GcpLogs calling close doesn't call
	// sync so we call Sync explicitly.
	if err := l.Sync(); err != nil {
		log.Error(err, "Error flushing logs")
		_, _ = fmt.Fprintf(os.Stdout, "Error flushing logs: %v\n", err)
	}
	for _, closer := range a.logClosers {
		closer()
	}
	return nil
}
