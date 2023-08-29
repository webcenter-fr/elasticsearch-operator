package main

import (
	"os"
	osruntime "runtime"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap/zapcore"
)

func getZapLogLevel() zapcore.Level {
	switch logLevel, _ := os.LookupEnv("LOG_LEVEL"); strings.ToLower(logLevel) {
	case zapcore.DebugLevel.String():
		return zapcore.DebugLevel
	case zapcore.InfoLevel.String():
		return zapcore.InfoLevel
	case zapcore.WarnLevel.String():
		return zapcore.WarnLevel
	case zapcore.ErrorLevel.String():
		return zapcore.ErrorLevel
	case zapcore.PanicLevel.String():
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

func getLogrusLogLevel() logrus.Level {
	switch logLevel, _ := os.LookupEnv("LOG_LEVEL"); strings.ToLower(logLevel) {
	case logrus.DebugLevel.String():
		return logrus.DebugLevel
	case logrus.InfoLevel.String():
		return logrus.InfoLevel
	case logrus.WarnLevel.String():
		return logrus.WarnLevel
	case logrus.ErrorLevel.String():
		return logrus.ErrorLevel
	case logrus.PanicLevel.String():
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

func getLogrusFormatter() logrus.Formatter {
	logFormatter, _ := os.LookupEnv("LOG_FORMATTER")
	if logFormatter == "json" {
		return &logrus.JSONFormatter{}
	}

	return &logrus.TextFormatter{}
}

func printVersion(logger logr.Logger, metricsAddr, probeAddr string) {
	logger.Info("Binary info ", "Go version", osruntime.Version())
	logger.Info("Binary info ", "OS", osruntime.GOOS, "Arch", osruntime.GOARCH)
	logger.Info("Address ", "Metrics", metricsAddr)
	logger.Info("Address ", "Probe", probeAddr)
}

func getWatchNamespace() (ns string, err error) {

	watchNamespaceEnvVar := "WATCH_NAMESPACES"
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", errors.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}

func getKubeClientTimeout() (timeout time.Duration, err error) {
	kubeClientTimeoutEnvVar := "KUBE_CLIENT_TIMEOUT"
	t, found := os.LookupEnv(kubeClientTimeoutEnvVar)
	if !found {
		return 30 * time.Second, nil
	}

	timeout, err = time.ParseDuration(t)
	if err != nil {
		return 0, err
	}

	return timeout, nil
}
