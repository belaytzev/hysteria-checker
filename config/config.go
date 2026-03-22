package config

import (
	"fmt"
	"time"

	"github.com/alecthomas/kong"
)

// Config holds the application configuration, parsed from CLI flags and environment variables.
type Config struct {
	SubscriptionURL []string      `kong:"name='subscription-url',env='SUBSCRIPTION_URL',help='Subscription URLs or direct hysteria:// / hysteria2:// / hy2:// links (comma-separated or repeated).'"`
	CheckInterval   time.Duration `kong:"name='check-interval',env='CHECK_INTERVAL',default='300s',help='Interval between health checks.'"`
	CheckMethod     string        `kong:"name='check-method',env='CHECK_METHOD',default='ip',enum='ip,status',help='Check method: ip or status.'"`
	CheckURL        string        `kong:"name='check-url',env='CHECK_URL',default='https://api.ipify.org',help='URL used for IP-based or status-based check.'"`
	CheckTimeout    time.Duration `kong:"name='check-timeout',env='CHECK_TIMEOUT',default='30s',help='Timeout for each proxy check.'"`
	MetricsHost     string        `kong:"name='metrics-host',env='METRICS_HOST',default='0.0.0.0',help='Host to bind the metrics/web server.'"`
	MetricsPort     int           `kong:"name='metrics-port',env='METRICS_PORT',default='2112',help='Port for the metrics/web server.'"`
	MetricsProtected bool        `kong:"name='metrics-protected',env='METRICS_PROTECTED',default='false',help='Enable Basic Auth for metrics.'"`
	MetricsUsername  string       `kong:"name='metrics-username',env='METRICS_USERNAME',help='Basic Auth username for metrics.'"`
	MetricsPassword  string       `kong:"name='metrics-password',env='METRICS_PASSWORD',help='Basic Auth password for metrics.'"`
	RedactSensitive bool          `kong:"name='web-public',env='WEB_PUBLIC',default='false',help='If true, redact sensitive details (server addresses, errors) in the dashboard and API responses.'"`
	LogLevel        string        `kong:"name='log-level',env='LOG_LEVEL',default='info',enum='debug,info,warn,error',help='Log level.'"`
}

// Parse parses the config from CLI args and environment variables.
func Parse(args []string) (*Config, error) {
	var cfg Config
	parser, err := kong.New(&cfg, kong.Name("hysteria-checker"), kong.Description("Hysteria proxy health checker"))
	if err != nil {
		return nil, err
	}
	_, err = parser.Parse(args)
	if err != nil {
		return nil, err
	}
	if cfg.CheckInterval <= 0 {
		return nil, fmt.Errorf("check-interval must be a positive duration, got %v", cfg.CheckInterval)
	}
	if cfg.MetricsPort < 1 || cfg.MetricsPort > 65535 {
		return nil, fmt.Errorf("metrics-port must be between 1 and 65535, got %d", cfg.MetricsPort)
	}
	if cfg.CheckTimeout <= 0 {
		return nil, fmt.Errorf("check-timeout must be a positive duration, got %v", cfg.CheckTimeout)
	}
	if cfg.MetricsProtected && (cfg.MetricsUsername == "" || cfg.MetricsPassword == "") {
		return nil, fmt.Errorf("metrics-username and metrics-password must be set when metrics-protected is enabled")
	}
	return &cfg, nil
}
