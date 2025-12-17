// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package witchcraft

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"maps"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-client/httpclient"
	"github.com/palantir/go-encrypted-config-value/encryptedconfigvalue"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/pkg/signals"
	werror "github.com/palantir/witchcraft-go-error"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/extractor"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/status"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/internal/dependencyhealth"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/internal/middleware"
	refreshablehealth "github.com/palantir/witchcraft-go-server/v3/witchcraft/internal/refreshable"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/wdebug"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	"github.com/palantir/witchcraft-go-server/v3/wrouter/whttprouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wzipkin"
	"gopkg.in/yaml.v2"
	yamlv3 "gopkg.in/yaml.v3"

	// Use zap as logger implementation: witchcraft-based applications are opinionated about the logging implementation used
	_ "github.com/palantir/witchcraft-go-logging/wlog-zap"
)

type Server[I config.BaseInstallConfig, R config.BaseRuntimeConfig] struct {
	// handlers specifies any custom HTTP handlers that should be used by the server. The provided handlers are invoked
	// in order after the built-in handlers (which provide things such as panic handling). The context in the request
	// will have the appropriate loggers and logger parameters set.
	handlers []wrouter.RequestHandlerMiddleware

	// useSelfSignedServerCertificate specifies whether the server uses a dynamically generated self-signed certificate
	// for TLS. No verification mechanism is provided for the self-signed certificate, so clients can only connect to a
	// server using this mode in an untrusted manner. As such, this option should only be used in very specialized
	// scenarios such as tests or in an environment where the server is exposed in a way that the connection to it can
	// be trusted based on other external mechanisms (in the latter scenario, using HTTPS with an unverified certificate
	// still provides the benefit that the traffic itself is encrypted).
	//
	// If false, the key material at the paths specified in serverConfig.CertFile and serverConfig.KeyFile is used.
	useSelfSignedServerCertificate bool

	// manages storing and retrieving server state (idle, initializing, running)
	stateManager serverStateManager

	// specifies the io.Writer to which goroutine dump will be written if a SIGQUIT is received while the server is
	// running. If nil, os.Stdout is used as the default. If the value is io.Discard, then no plaintext output will
	// be emitted. A diagnostic.1 line is logged unless disableSigQuitHandler is true.
	sigQuitHandlerWriter io.Writer

	// if true, disables the default behavior of emitting a goroutine dump on SIGQUIT signals.
	disableSigQuitHandler bool

	// if true, disables the default behavior of shutting down the server on SIGTERM and SIGINT signals.
	disableShutdownSignalHandler bool

	// provides the bytes for the install configuration for the server. If nil, a default configuration provider that
	// reads the file at "var/conf/install.yml" is used.
	installConfigProvider ConfigBytesProvider

	// a function that provides the refreshable.Validated that provides the bytes for the runtime configuration for
	// the server. The ctx provided to the function is valid for the lifetime of the server. If nil, uses a function
	// that returns a default file-based Refreshable that reads the file at "var/conf/runtime.yml". The value of the
	// Refreshable is "[]byte", where the byte slice is the contents of the runtime configuration file.
	//
	// The returned refreshable.Validated[[]byte] tracks validation state, which includes both the ability to read the
	// configuration file and to unmarshal it into the expected type R. Validation failures are exposed via
	// the CONFIG_RELOAD health check source. When validation succeeds after a previous failure, the server
	// automatically uses the new valid configuration.
	runtimeConfigProvider func(ctx context.Context) refreshable.Validated[[]byte]

	// specifies the source used to provide the readiness information for the server. If nil, a default value that uses
	// the server's status is used.
	readinessSource healthstatus.Source

	// specifies the source used to provide the liveness information for the server. If nil, a default value that uses
	// the server's status is used.
	livenessSource healthstatus.Source

	// specifies the sources that are used to determine the health of this service
	healthCheckSources []healthstatus.HealthCheckSource

	// specifies the handlers to invoke upon health status changes. The LoggingHealthStatusChangeHandler is added by default.
	healthStatusChangeHandlers []status.HealthStatusChangeHandler

	customDiagnosticHandlers []wdebug.DiagnosticHandler

	// if true, disables the SERVICE_DEPENDENCY health check.
	disableServiceDependencyHealth bool

	// provides the SERVICE_DEPENDENCY health check unless disableServiceDependencyHealth is true.
	serviceDependencyHealthCheck *dependencyhealth.ServiceDependencyHealthCheck

	// provides the RouterImpl used by the server (and management server if it is separate). If nil, a default function
	// that returns a new whttprouter is used.
	routerImplProvider func() wrouter.RouterImpl

	// called on server initialization before the server starts. Is provided with a context that is active for the
	// duration of the server lifetime, the server router (which can be used to register endpoints), the unmarshaled
	// install configuration and the refreshable runtime configuration.
	//
	// If this function returns an error, the server is not started and the error is returned.
	initFn InitFunc[I, R]

	// provides the encrypted-config-value key that is used to decrypt encrypted values in configuration. If nil, a
	// default provider that reads the key from the file at "var/conf/encrypted-config-value.key" is used.
	ecvKeyProvider ECVKeyProvider

	// if true, then Go runtime metrics will not be recorded. If false, Go runtime metrics will be recorded at a
	// collection interval that matches the metric emit interval specified in the install configuration (or every 60
	// seconds if an interval is not specified in configuration).
	disableGoRuntimeMetrics bool

	// metricsBlocklist specifies the set of metrics that should not be emitted by the metric logger.
	metricsBlocklist map[string]struct{}

	// metricTypeValuesBlocklist specifies the values for a metric type that should be omitted from metric output. For
	// example, if the map is set to {"timer":{"5m":{}}}, then the value for "5m" will be omitted from all timer metric
	// output. If nil, the default value is the map returned by defaultMetricTypeValuesBlocklist().
	metricTypeValuesBlocklist map[string]map[string]struct{}

	// endpoint500sHealthCheckFunc builds the ENDPOINT_FIVE_HUNDREDS health check source. If nil, the check is disabled.
	// The health check source is enabled by default.
	// Use WithDisableEndpointFiveHundredsHealthCheck to disable the health check.
	// Use WithAlwaysHealthyEndpointFiveHundredsHealthCheck to always report the health check as healthy.
	endpoint500sHealthCheckFunc func(ctx context.Context) *middleware.EndpointFiveHundredsHealthCheck

	// specifies the TLS client authentication mode used by the server. If not specified, the default value is
	// tls.NoClientCert.
	clientAuth tls.ClientAuthType

	// specifies the value used for the "origin" field for service logs. If not specified, the default value is set to
	// be the package from which "Start" was called.
	svcLogOrigin *string

	// specifies that the service.1 logger should use the call site for the origin field.
	// See docs on svc1log.OriginFromCallLine for details.
	svcLogOriginFromCallLine bool

	// applicationTraceSampler is the function that is used to determine whether or not a trace should be sampled.
	// This applies to routes registered under the application port and the context passed to the initialize function
	// If nil, the default behavior is to sample every trace.
	applicationTraceSampler wtracing.Sampler

	// managementTraceSampler is the function that is used to determine whether or not a trace should be sampled.
	// This applies to routes registered under the management port
	// If nil, the default behavior is to sample no traces.
	managementTraceSampler wtracing.Sampler

	// disableKeepAlives disables keep-alives.
	disableKeepAlives bool

	// disableHTTP2 disables HTTP/2 support.
	disableHTTP2 bool

	// configYAMLUnmarshalFn is the function used to unmarshal YAML configuration. By default, this is yaml.Unmarshal.
	configYAMLUnmarshalFn func(in []byte, out interface{}) (err error)

	// request logger configuration

	// idsExtractor specifies the extractor used to extract identifiers (such as UID, SID, TokenID) from requests for
	// request logging and middleware. If nil, uses extractor.NewDefaultIDsExtractor().
	idsExtractor     extractor.IDsFromRequest
	safePathParams   []string
	safeQueryParams  []string
	safeHeaderParams []string

	// loggerStdoutWriter specifies the io.Writer that is written to if the loggers are in a mode that specifies that
	// they should write to Stdout. If nil, os.Stdout is used by default.
	loggerStdoutWriter io.Writer

	// Parameters that control dual-output of audit logging.
	// Currently, the default behavior is to not dual-log.

	// if true, initializes loggers such that any entry written to the "audit.2" logger
	// is also dual-logged to the "audit.3" logger.
	dualLogAuditV2ToAuditV3 bool

	// if true, initializes loggers such that any entry written to the "audit.3" logger
	// is also dual-logged to the "audit.2" logger.
	dualLogAuditV3ToAuditV2 bool

	// loggers
	svcLogger    svc1log.Logger
	evtLogger    evt2log.Logger
	audit2Logger audit2log.Logger
	// stored as an atomic pointer rather than a logger directly because the value may be updated (common parameter
	// values can be specified in reloadable runtime configuration)
	audit3Logger atomic.Pointer[audit3log.Logger]
	metricLogger metric1log.Logger
	trcLogger    trc1log.Logger
	diagLogger   diag1log.Logger
	reqLogger    req2log.Logger

	// the http.Server for the main server
	httpServer *http.Server

	// allows the server to wait until Close() or Shutdown() return prior to returning from Start()
	shutdownFinished chan struct{}
}

// InitFunc is a function type used to initialize a server. ctx is a context configured with loggers and is valid for
// the duration of the server. Refer to the documentation of InitInfo for its fields.
//
// If the returned cleanup function is non-nil, it is deferred and run on server shutdown. If the returned error is
// non-nil, the server will not start and will return the error.
type InitFunc[I config.BaseInstallConfig, R config.BaseRuntimeConfig] func(ctx context.Context, info InitInfo[I, R]) (cleanup func(), rErr error)

type InitInfo[I config.BaseInstallConfig, R config.BaseRuntimeConfig] struct {
	// Router is a ConfigurableRouter that implements wrouter.Router for the server. It can be
	// used to register endpoints on the server and to configure things such as health, readiness and liveness sources and
	// any middleware (note that any values set using router will override any values previously set on the server).
	Router ConfigurableRouter[I, R]

	// InstallConfig is the install configuration.
	InstallConfig I

	// RuntimeConfig is a refreshable that contains the runtime configuration.
	RuntimeConfig refreshable.Refreshable[R]

	// Clients exposes the service-discovery configuration as a conjure-go-runtime client builder.
	// Returned clients are configured with user-agent based on {install.ProductName}/{install.ProductVersion}.
	Clients ConfigurableServiceDiscovery

	// ShutdownServer gracefully closes the server, waiting for any in-flight requests to finish (or the context to be cancelled).
	// When the InitFunc is executed, the server is not yet started. This will most often be useful if launching a goroutine which
	// requires access to shut down the server in some error condition.
	ShutdownServer func(context.Context) error
}

// ConfigurableRouter is a wrouter.Router that provides additional support for configuring things such as health,
// readiness, liveness and middleware.
type ConfigurableRouter[I config.BaseInstallConfig, R config.BaseRuntimeConfig] interface {
	wrouter.Router

	WithHealth(healthSources ...healthstatus.HealthCheckSource) *Server[I, R]
	WithReadiness(readiness healthstatus.Source) *Server[I, R]
	WithLiveness(liveness healthstatus.Source) *Server[I, R]
	WithCustomDiagnosticHandlers(handlers ...wdebug.DiagnosticHandler) *Server[I, R]
}

const defaultSampleRate = 0.01

// NewServer returns a new uninitialized server.
func NewServer[I config.BaseInstallConfig, R config.BaseRuntimeConfig]() *Server[I, R] {
	return &Server[I, R]{}
}

// WithInitFunc configures the server to use the provided setup function to set up its initial state.
func (s *Server[I, R]) WithInitFunc(initFn InitFunc[I, R]) *Server[I, R] {
	s.initFn = initFn
	return s
}

// WithInstallConfig configures the server to use the provided install configuration. The provided install configuration
// must support being marshaled as YAML.
func (s *Server[I, R]) WithInstallConfig(installConfigStruct I) *Server[I, R] {
	s.installConfigProvider = cfgBytesProviderFn(func() ([]byte, error) {
		return yaml.Marshal(installConfigStruct)
	})
	return s
}

// WithInstallConfigFromFile configures the server to read the install configuration from the file at the specified
// path.
func (s *Server[I, R]) WithInstallConfigFromFile(fpath string) *Server[I, R] {
	s.installConfigProvider = cfgBytesProviderFn(func() ([]byte, error) {
		return os.ReadFile(fpath)
	})
	return s
}

// WithInstallConfigProvider configures the server to use the install configuration obtained by reading the bytes from
// the specified ConfigBytesProvider.
func (s *Server[I, R]) WithInstallConfigProvider(p ConfigBytesProvider) *Server[I, R] {
	s.installConfigProvider = p
	return s
}

// WithRuntimeConfig configures the server to use the provided runtime configuration. The provided runtime configuration
// must support being marshaled as YAML.
func (s *Server[I, R]) WithRuntimeConfig(in R) *Server[I, R] {
	s.runtimeConfigProvider = func(_ context.Context) refreshable.Validated[[]byte] {
		v, _, _ := refreshable.MapWithError(refreshable.New(in), func(in R) ([]byte, error) {
			return yaml.Marshal(in)
		})
		return v
	}
	return s
}

// WithRuntimeConfigProvider configures the server to use the provided Refreshable as its runtime configuration. The
// value provided by the refreshable must be the byte slice for the runtime configuration.
func (s *Server[I, R]) WithRuntimeConfigProvider(r refreshable.Refreshable[[]byte]) *Server[I, R] {
	s.runtimeConfigProvider = func(context.Context) refreshable.Validated[[]byte] {
		if v, ok := r.(refreshable.Validated[[]byte]); ok {
			return v
		}
		v, _, _ := refreshable.Validate(r, func([]byte) error { return nil })
		return v
	}
	return s
}

// WithRuntimeConfigProviderFunc configures the server to use the returned Refreshable as its runtime configuration.
// The value provided by the refreshable must be a []byte for the yaml runtime configuration.
func (s *Server[I, R]) WithRuntimeConfigProviderFunc(f func(ctx context.Context) refreshable.Validated[[]byte]) *Server[I, R] {
	s.runtimeConfigProvider = f
	return s
}

// WithRuntimeConfigFromFile configures the server to use the file at the provided path as its runtime configuration.
// The server will create a refreshable.Refreshable using the file at the provided path (and will thus live-reload the
// configuration based on updates to the file).
func (s *Server[I, R]) WithRuntimeConfigFromFile(fpath string) *Server[I, R] {
	s.runtimeConfigProvider = func(ctx context.Context) refreshable.Validated[[]byte] {
		return refreshable.NewFileRefreshable(ctx, fpath)
	}
	return s
}

// WithSelfSignedCertificate configures the server to use a dynamically generated self-signed certificate for its TLS
// authentication. Because there is no way to verify the certificate used by the server, this option is typically only
// used in tests or very specialized circumstances where the connection to the server can be verified/authenticated
// using separate external mechanisms.
func (s *Server[I, R]) WithSelfSignedCertificate() *Server[I, R] {
	s.useSelfSignedServerCertificate = true
	return s
}

// WithECVKeyFromFile configures the server to use the ECV key in the file at the specified path as the ECV key for
// decrypting ECV values in configuration.
func (s *Server[I, R]) WithECVKeyFromFile(fPath string) *Server[I, R] {
	s.ecvKeyProvider = ECVKeyFromFile(fPath)
	return s
}

// WithECVKeyProvider configures the server to use the ECV key provided by the specified provider as the ECV key for
// decrypting ECV values in configuration.
func (s *Server[I, R]) WithECVKeyProvider(ecvProvider ECVKeyProvider) *Server[I, R] {
	s.ecvKeyProvider = ecvProvider
	return s
}

// WithClientAuth configures the server to use the specified client authentication type for its TLS connections.
func (s *Server[I, R]) WithClientAuth(clientAuth tls.ClientAuthType) *Server[I, R] {
	s.clientAuth = clientAuth
	return s
}

// WithHealth configures the server to use the specified health check sources to report the server's health. If multiple
// healthSource's results have the same key, the result from the latest entry in healthSources will be used. These
// results are combined with the server's built-in health source, which uses the `SERVER_STATUS` key.
func (s *Server[I, R]) WithHealth(healthSources ...healthstatus.HealthCheckSource) *Server[I, R] {
	s.healthCheckSources = healthSources
	return s
}

// WithReadiness configures the server to use the specified source to report readiness.
func (s *Server[I, R]) WithReadiness(readiness healthstatus.Source) *Server[I, R] {
	s.readinessSource = readiness
	return s
}

// WithLiveness configures the server to use the specified source to report liveness.
func (s *Server[I, R]) WithLiveness(liveness healthstatus.Source) *Server[I, R] {
	s.livenessSource = liveness
	return s
}

// WithOrigin configures the server to use the specified origin.
func (s *Server[I, R]) WithOrigin(origin string) *Server[I, R] {
	s.svcLogOrigin = &origin
	return s
}

// WithOriginFromCallLine configures the server to use the svc1log.OriginFromCallLine parameter.
func (s *Server[I, R]) WithOriginFromCallLine() *Server[I, R] {
	s.svcLogOriginFromCallLine = true
	return s
}

// WithMiddleware configures the server to use the specified middleware. The provided middleware is added to any other
// specified middleware.
func (s *Server[I, R]) WithMiddleware(middleware wrouter.RequestHandlerMiddleware) *Server[I, R] {
	s.handlers = append(s.handlers, middleware)
	return s
}

// WithRouterImplProvider configures the server to use the specified routerImplProvider to provide router
// implementations.
func (s *Server[I, R]) WithRouterImplProvider(routerImplProvider func() wrouter.RouterImpl) *Server[I, R] {
	s.routerImplProvider = routerImplProvider
	return s
}

// WithTraceSampler configures the server's application trace log tracer to use the specified traceSampler function to make a
// determination on whether or not a trace should be sampled (if such a decision needs to be made).
func (s *Server[I, R]) WithTraceSampler(traceSampler wtracing.Sampler) *Server[I, R] {
	s.applicationTraceSampler = traceSampler
	return s
}

// WithTraceSamplerRate is a convenience function for creating an application traceSampler based off a sample rate
func (s *Server[I, R]) WithTraceSamplerRate(sampleRate float64) *Server[I, R] {
	return s.WithTraceSampler(traceSamplerFromSampleRate(sampleRate))
}

// WithManagementTraceSampler configures the server's management trace log tracer to use the specified traceSampler function to make a
// determination on whether or not a trace should be sampled (if such a decision needs to be made).
func (s *Server[I, R]) WithManagementTraceSampler(traceSampler wtracing.Sampler) *Server[I, R] {
	s.managementTraceSampler = traceSampler
	return s
}

// WithManagementTraceSamplerRate is a convenience function for creating a management traceSampler based off a sample rate
func (s *Server[I, R]) WithManagementTraceSamplerRate(sampleRate float64) *Server[I, R] {
	return s.WithManagementTraceSampler(traceSamplerFromSampleRate(sampleRate))
}

// WithSigQuitHandlerWriter sets the output for the goroutine dump on SIGQUIT.
func (s *Server[I, R]) WithSigQuitHandlerWriter(w io.Writer) *Server[I, R] {
	s.sigQuitHandlerWriter = w
	return s
}

// WithDisableSigQuitHandler disables the server's enabled-by-default goroutine dump on SIGQUIT.
func (s *Server[I, R]) WithDisableSigQuitHandler() *Server[I, R] {
	s.disableSigQuitHandler = true
	return s
}

// WithDisableShutdownSignalHandler disables the server's enabled-by-default shutdown on SIGTERM and SIGINT.
func (s *Server[I, R]) WithDisableShutdownSignalHandler() *Server[I, R] {
	s.disableShutdownSignalHandler = true
	return s
}

// WithDisableKeepAlives disables keep-alives on the server by calling SetKeepAlivesEnabled(false) on the http.Server
// used by the server. Note that this setting is only applied to the main server -- if the management server is separate
// from the main server, this setting is not applied to the management server. Refer to the documentation for
// SetKeepAlivesEnabled in http.Server for more information on when a server may want to use this setting.
func (s *Server[I, R]) WithDisableKeepAlives() *Server[I, R] {
	s.disableKeepAlives = true
	return s
}

// WithDisableHTTP2 disables HTTP/2 support on the server by setting TLSNextProto to an empty map on the http.Server.
// Note that this setting is only applied to the main server. You should only disable HTTP/2 if you are using handlers
// that require HTTP/1, such as WebSockets.
func (s *Server[I, R]) WithDisableHTTP2() *Server[I, R] {
	s.disableHTTP2 = true
	return s
}

// WithDisableGoRuntimeMetrics disables the server's enabled-by-default collection of runtime memory statistics.
func (s *Server[I, R]) WithDisableGoRuntimeMetrics() *Server[I, R] {
	s.disableGoRuntimeMetrics = true
	return s
}

// WithDisableServiceDependencyHealth disables the server's enabled-by-default SERVICE_DEPENDENCY check.
func (s *Server[I, R]) WithDisableServiceDependencyHealth() *Server[I, R] {
	s.disableServiceDependencyHealth = true
	return s
}

// WithMetricsBlocklist sets the metric blocklist to the provided set of metrics. The provided metrics should be the
// name of the metric (for example, "server.response.size"). The blocklist only supports blocklisting at the metric
// level: blocklisting an individual metric value (such as "server.response.size.count") will not have any effect. The
// provided input is copied.
func (s *Server[I, R]) WithMetricsBlocklist(blocklist map[string]struct{}) *Server[I, R] {
	metricsBlocklist := make(map[string]struct{})
	maps.Copy(metricsBlocklist, blocklist)
	s.metricsBlocklist = metricsBlocklist
	return s
}

// WithMetricTypeValuesBlocklist sets the value of the metric type value blocklist to be the same as the provided value
// (the content is copied).
func (s *Server[I, R]) WithMetricTypeValuesBlocklist(blocklist map[string]map[string]struct{}) *Server[I, R] {
	newBlocklist := make(map[string]map[string]struct{}, len(blocklist))
	for k, v := range blocklist {
		newVal := make(map[string]struct{}, len(v))
		for kk := range v {
			newVal[kk] = struct{}{}
		}
		newBlocklist[k] = newVal
	}
	s.metricTypeValuesBlocklist = newBlocklist
	return s
}

// WithDisableEndpointFiveHundredsHealthCheck disables the ENDPOINT_FIVE_HUNDREDS healthcheck.
func (s *Server[I, R]) WithDisableEndpointFiveHundredsHealthCheck() *Server[I, R] {
	s.endpoint500sHealthCheckFunc = func(ctx context.Context) *middleware.EndpointFiveHundredsHealthCheck {
		return nil
	}
	return s
}

// WithAlwaysHealthyEndpointFiveHundredsHealthCheck configures the server to always report the ENDPOINT_FIVE_HUNDREDS
// as healthy, even if the parameters include failing or broken endpoints.
func (s *Server[I, R]) WithAlwaysHealthyEndpointFiveHundredsHealthCheck() *Server[I, R] {
	s.endpoint500sHealthCheckFunc = func(ctx context.Context) *middleware.EndpointFiveHundredsHealthCheck {
		return middleware.NewEndpointFiveHundredsHealthCheck(ctx, true)
	}
	return s
}

// WithLoggerStdoutWriter configures the writer that loggers will write to IF they are configured to write to STDOUT.
// This configuration is typically only used in specialized scenarios (for example, to write logger output to an
// in-memory buffer rather than Stdout for tests).
func (s *Server[I, R]) WithLoggerStdoutWriter(loggerStdoutWriter io.Writer) *Server[I, R] {
	s.loggerStdoutWriter = loggerStdoutWriter
	return s
}

// WithHealthStatusChangeHandlers configures the health status change handlers that are called whenever the configured HealthCheckSource
// returns a health status with differing check states.
func (s *Server[I, R]) WithHealthStatusChangeHandlers(handlers ...status.HealthStatusChangeHandler) *Server[I, R] {
	s.healthStatusChangeHandlers = append(s.healthStatusChangeHandlers, handlers...)
	return s
}

// WithCustomDiagnosticHandlers configures the application's custom diagnostic handlers.
// This adds to the default diagnostic handlers provided by the server.
func (s *Server[I, R]) WithCustomDiagnosticHandlers(handlers ...wdebug.DiagnosticHandler) *Server[I, R] {
	s.customDiagnosticHandlers = append(s.customDiagnosticHandlers, handlers...)
	return s
}

// WithEnableDualLogAuditV2ToAuditV3 enables dual-writing audit v2 logs to audit v3 logs.
// This is an experimental feature: the functionality or function itself may be removed in the future.
func (s *Server[I, R]) WithEnableDualLogAuditV2ToAuditV3() *Server[I, R] {
	s.dualLogAuditV2ToAuditV3 = true
	return s
}

// WithEnableDualLogAuditV3ToAuditV2 enables dual-writing audit v3 logs to audit v2 logs.
// This is an experimental feature: the functionality or function itself may be removed in the future.
func (s *Server[I, R]) WithEnableDualLogAuditV3ToAuditV2() *Server[I, R] {
	s.dualLogAuditV3ToAuditV2 = true
	return s
}

const (
	defaultMetricEmitFrequency = time.Second * 30

	ecvKeyPath        = "var/conf/encrypted-config-value.key"
	installConfigPath = "var/conf/install.yml"
	runtimeConfigPath = "var/conf/runtime.yml"

	runtimeConfigReloadCheckType = "CONFIG_RELOAD"
)

// Start begins serving HTTPS traffic and blocks until s.Close() or s.Shutdown() return.
// Errors are logged via s.svcLogger before being returned.
// Panics are recovered; in the case of a recovered panic, Start will log and return
// a non-nil error containing the recovered object (overwriting any existing error).
func (s *Server[I, R]) Start() (rErr error) {

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				rErr = err
			} else {
				rErr = werror.Error("panic recovered", werror.UnsafeParam("recovered", r))
			}

			if s.svcLogger == nil {
				// If we have not yet initialized our loggers, use default configuration as best-effort.
				s.initDefaultLoggers(false, wlog.InfoLevel, metrics.DefaultMetricsRegistry)
			}

			s.svcLogger.Error("panic recovered", svc1log.SafeParam("stack", diag1log.ThreadDumpV1FromGoroutines(debug.Stack())), svc1log.Stacktrace(rErr))
		}
	}()
	defer func() {
		if rErr != nil {
			if s.svcLogger == nil {
				// If we have not yet initialized our loggers, use default configuration as best-effort.
				s.initDefaultLoggers(false, wlog.InfoLevel, metrics.DefaultMetricsRegistry)
			}
			// safelogging:@Allow: kept for backwards compatibility, but consider updating
			s.svcLogger.Error(rErr.Error(), svc1log.Stacktrace(rErr))
		}
	}()

	// Set state to "initializing". Fails if current state is not "idle" (ensures that this instance is not being run
	// concurrently).
	if err := s.stateManager.Start(); err != nil {
		return err
	}
	// Channel can be nil if this is the first Start() or closed if the previous run ended with a Close() or Shutdown().
	// Since stateManager.Start() succeeded, we know that no shutdown of a previous run is ongoing.
	s.shutdownFinished = make(chan struct{})
	// Ensure that state is reset to "ServerIdle" before Start() returns.
	defer func() {
		curState := s.State()
		for {
			switch curState {
			case ServerIdle:
				return
			case ServerShuttingDown:
				// Wait for s.Close() or s.Shutdown() to return if called.
				// Once the below channel is closed, the state is guaranteed to be "ServerIdle".
				<-s.shutdownFinished
			default:
				if s.stateManager.compareAndSwapState(curState, ServerIdle) {
					return
				}
			}
			curState = s.State()
		}
	}()

	// set provider for ECV key
	if s.ecvKeyProvider == nil {
		s.ecvKeyProvider = ECVKeyFromFile(ecvKeyPath)
	}

	// if config unmarshal function is not set, default to yaml.Unmarshal
	if s.configYAMLUnmarshalFn == nil {
		s.configYAMLUnmarshalFn = yaml.Unmarshal
	}

	// load install configuration
	fullInstallCfg, err := s.initInstallConfig()
	if err != nil {
		return err
	}
	baseInstallCfg := fullInstallCfg.BaseInstallConfig()

	if s.idsExtractor == nil {
		s.idsExtractor = extractor.NewDefaultIDsExtractor()
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// initialize metrics. Note that loggers have not been initialized or associated with ctx
	metricsRegistry, metricsDeferFn, err := s.initMetrics(ctx, baseInstallCfg)
	if err != nil {
		return err
	}
	defer metricsDeferFn()
	ctx = metrics.WithRegistry(ctx, metricsRegistry)

	// initialize loggers
	if baseInstallCfg.UseWrappedLogs {
		s.initWrappedLoggers(baseInstallCfg.UseConsoleLog, baseInstallCfg.ProductName, baseInstallCfg.ProductVersion, wlog.InfoLevel, metricsRegistry)
	} else {
		s.initDefaultLoggers(baseInstallCfg.UseConsoleLog, wlog.InfoLevel, metricsRegistry)
	}

	// add loggers to context
	ctx = s.withLoggers(ctx)

	// load runtime configuration
	refreshableRuntimeCfg, configReloadHealthCheckSource, err := s.initRuntimeConfig(ctx)
	if err != nil {
		return err
	}
	internalHealthCheckSources := []healthstatus.HealthCheckSource{configReloadHealthCheckSource}

	// set up SERVICE_DEPENDENCY check
	if !s.disableServiceDependencyHealth {
		s.serviceDependencyHealthCheck = dependencyhealth.NewServiceDependencyHealthCheck()
		internalHealthCheckSources = append(internalHealthCheckSources, s.serviceDependencyHealthCheck)
	}

	// Set the service log level if configured
	unsubscribe := refreshableRuntimeCfg.Subscribe(func(r R) {
		if loggerCfg := r.BaseRuntimeConfig().LoggerConfig; loggerCfg != nil && loggerCfg.Level != "" {
			s.svcLogger.SetLevel(loggerCfg.Level)
		}
	})
	defer unsubscribe()

	// Set audit logger configuration
	auditLoggerConfig, unsubscribe := refreshable.Map(refreshableRuntimeCfg, func(r R) *config.AuditConfig { return r.BaseRuntimeConfig().AuditConfig })
	defer unsubscribe()
	unsubscribe = auditLoggerConfig.Subscribe(s.updateAuditLoggerConfig)
	defer unsubscribe()
	ctx = audit3log.WithLogger(ctx, *s.audit3Logger.Load())

	if s.routerImplProvider == nil {
		s.routerImplProvider = func() wrouter.RouterImpl {
			return whttprouter.New()
		}
	}

	var endpoint500s *middleware.EndpointFiveHundredsHealthCheck
	if s.endpoint500sHealthCheckFunc != nil {
		endpoint500s = s.endpoint500sHealthCheckFunc(ctx)
	} else {
		endpoint500s = middleware.NewEndpointFiveHundredsHealthCheck(ctx, false)
	}
	internalHealthCheckSources = append(internalHealthCheckSources, endpoint500s)

	// initialize routers
	router, mgmtRouter := s.initRouters(baseInstallCfg)

	// add middleware
	s.addMiddleware(router.RootRouter(), metricsRegistry, endpoint500s, s.getApplicationTracingOptions(baseInstallCfg))
	if mgmtRouter != router {
		// add middleware to management router as well if it is distinct
		s.addMiddleware(mgmtRouter.RootRouter(), metricsRegistry, endpoint500s, s.getManagementTracingOptions(baseInstallCfg))
		// add debugging endpoints to management router
		if err := addPprofRoutes(mgmtRouter); err != nil {
			return werror.Wrap(err, "failed to register debugging routes")
		}
	}

	s.initStackTraceHandler(ctx)
	s.initShutdownSignalHandler(ctx)

	if s.initFn != nil {
		traceReporter := wtracing.NewNoopReporter()
		if s.trcLogger != nil {
			traceReporter = s.trcLogger
		}
		tracer, err := wzipkin.NewTracer(traceReporter, s.getApplicationTracingOptions(baseInstallCfg)...)
		if err != nil {
			return err
		}
		ctx = wtracing.ContextWithTracer(ctx, tracer)

		refreshableServicesConfig, _ := refreshable.Map(refreshableRuntimeCfg, func(t R) httpclient.ServicesConfig {
			return t.BaseRuntimeConfig().ServiceDiscovery
		})
		discovery := NewServiceDiscovery(baseInstallCfg, refreshableServicesConfig)
		if s.serviceDependencyHealthCheck != nil {
			discovery.WithDefaultParams(func(serviceName string) ([]httpclient.ClientParam, error) {
				return []httpclient.ClientParam{
					httpclient.WithMiddleware(s.serviceDependencyHealthCheck.Middleware(serviceName)),
				}, nil
			})
		}

		svc1log.FromContext(ctx).Debug("Running server initialization function.")
		cleanupFn, err := s.initFn(
			ctx,
			InitInfo[I, R]{
				Router: &configurableRouterImpl[I, R]{
					Router: newMultiRouterImpl(router, mgmtRouter),
					Server: s,
				},
				InstallConfig:  fullInstallCfg,
				RuntimeConfig:  refreshableRuntimeCfg,
				Clients:        discovery,
				ShutdownServer: s.Shutdown,
			},
		)
		if err != nil {
			return err
		}
		if cleanupFn != nil {
			defer cleanupFn()
		}
	}

	// add all internally defined health check sources to the user supplied ones after running the initFn.
	s.healthCheckSources = append(s.healthCheckSources, internalHealthCheckSources...)

	// add routes for health, liveness and readiness. Must be done after initFn to ensure that any
	// health/liveness/readiness configuration updated by initFn is applied.
	if err := s.addRoutes(ctx, mgmtRouter, refreshableRuntimeCfg); err != nil {
		return err
	}

	// only create and start a separate management http server if management port is explicitly specified and differs
	// from the main server port
	if mgmtPort := baseInstallCfg.Server.ManagementPort; mgmtPort != 0 && baseInstallCfg.Server.Port != mgmtPort {
		mgmtStart, mgmtShutdown, err := s.newMgmtServer(baseInstallCfg.ProductName, baseInstallCfg.Server, mgmtRouter.RootRouter())
		if err != nil {
			return err
		}

		// start management server in its own goroutine
		go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
			if err := mgmtStart(); err != nil {
				svc1log.FromContext(ctx).Error("management server failed", svc1log.Stacktrace(err))
			}
		})
		defer func() {
			if err := mgmtShutdown(ctx); err != nil {
				svc1log.FromContext(ctx).Error("management server failed to shutdown", svc1log.Stacktrace(err))
			}
		}()
	}

	httpServer, svrStart, _, err := s.newServer(baseInstallCfg.ProductName, baseInstallCfg.Server, router.RootRouter(), s.connStateCallback(ctx))
	if err != nil {
		return err
	}

	s.httpServer = httpServer
	if s.disableKeepAlives {
		s.httpServer.SetKeepAlivesEnabled(false)
	}

	if s.disableHTTP2 {
		s.httpServer.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}

	if !s.stateManager.compareAndSwapState(ServerInitializing, ServerRunning) {
		return werror.ErrorWithContextParams(ctx, "server was shut down before it could start")
	}
	return svrStart()
}

func (s *Server[I, R]) connStateCallback(ctx context.Context) func(conn net.Conn, state http.ConnState) {
	return func(conn net.Conn, state http.ConnState) {
		metrics.FromContext(ctx).Counter("server.conn_state_change", metrics.MustNewTag("state", state.String())).Inc(1)
	}
}

func (s *Server[I, R]) withLoggers(ctx context.Context) context.Context {
	ctx = svc1log.WithLogger(ctx, s.svcLogger)
	ctx = evt2log.WithLogger(ctx, s.evtLogger)
	ctx = metric1log.WithLogger(ctx, s.metricLogger)
	ctx = trc1log.WithLogger(ctx, s.trcLogger)
	ctx = audit2log.WithLogger(ctx, s.audit2Logger)
	ctx = audit3log.WithLogger(ctx, *s.audit3Logger.Load())
	ctx = diag1log.WithLogger(ctx, s.diagLogger)
	return ctx
}

type configurableRouterImpl[I config.BaseInstallConfig, R config.BaseRuntimeConfig] struct {
	wrouter.Router
	*Server[I, R]
}

func (s *Server[I, R]) initInstallConfig() (zero I, _ error) {
	if s.installConfigProvider == nil {
		// if install config provider is not specified, use a file-based one
		s.installConfigProvider = cfgBytesProviderFn(func() ([]byte, error) {
			return os.ReadFile(installConfigPath)
		})
	}

	cfgBytes, err := s.installConfigProvider.LoadBytes()
	if err != nil {
		return zero, werror.Wrap(err, "Failed to load install configuration bytes")
	}
	cfgBytes, err = s.decryptConfigBytes(cfgBytes)
	if err != nil {
		return zero, werror.Wrap(err, "Failed to decrypt install configuration bytes")
	}
	var installConfigStruct I
	if err := s.configYAMLUnmarshalFn(cfgBytes, &installConfigStruct); err != nil {
		return zero, werror.Wrap(err, "Failed to unmarshal install specific configuration YAML")
	}
	return installConfigStruct, nil
}

func (s *Server[I, R]) initRuntimeConfig(ctx context.Context) (rCfg refreshable.Refreshable[R], hcSrc healthstatus.HealthCheckSource, rErr error) {
	if s.runtimeConfigProvider == nil {
		// if runtime provider is not specified, use a file-based one
		s.runtimeConfigProvider = func(ctx context.Context) refreshable.Validated[[]byte] {
			return refreshable.NewFileRefreshable(ctx, runtimeConfigPath)
		}
	}

	runtimeConfigProvider := s.runtimeConfigProvider(ctx)
	if _, err := runtimeConfigProvider.Validation(); err != nil {
		return nil, nil, err
	}
	unmarshalledRuntimeConfig, _, err := refreshable.MapWithError(runtimeConfigProvider, func(cfgBytes []byte) (R, error) {
		cfgBytes, err := s.decryptConfigBytes(cfgBytes)
		if err != nil {
			s.svcLogger.Warn("Failed to decrypt encrypted runtime configuration", svc1log.Stacktrace(err))
		}
		var runtimeCfg R
		if err := s.configYAMLUnmarshalFn(cfgBytes, &runtimeCfg); err != nil {
			var zero R
			return zero, err
		}
		return runtimeCfg, nil
	})
	if err != nil {
		return nil, nil, err
	}

	validatingRefreshableHealthCheckSource := refreshablehealth.NewValidatingRefreshableHealthCheckSource(
		runtimeConfigReloadCheckType,
		refreshablehealth.ValidationErrFunc(runtimeConfigProvider),
		refreshablehealth.ValidationErrFunc(unmarshalledRuntimeConfig),
	)

	return unmarshalledRuntimeConfig, validatingRefreshableHealthCheckSource, nil
}

func (s *Server[I, R]) initStackTraceHandler(ctx context.Context) {
	if s.disableSigQuitHandler {
		return
	}

	stackTraceHandler := func(stackTraceOutput []byte) error {
		if s.diagLogger != nil {
			s.diagLogger.Diagnostic(logging.NewDiagnosticFromThreadDump(diag1log.ThreadDumpV1FromGoroutines(stackTraceOutput)))
		}
		if s.sigQuitHandlerWriter != nil {
			if _, err := s.sigQuitHandlerWriter.Write(stackTraceOutput); err != nil {
				return err
			}
		}
		return nil
	}
	errHandler := func(err error) {
		if s.svcLogger != nil && err != nil {
			s.svcLogger.Error("Failed to dump goroutines", svc1log.Stacktrace(err))
		}
	}

	signals.RegisterStackTraceHandlerOnSignals(ctx, stackTraceHandler, errHandler, syscall.SIGQUIT)
}

func (s *Server[I, R]) initShutdownSignalHandler(ctx context.Context) {
	if s.disableShutdownSignalHandler {
		return
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGTERM, syscall.SIGINT)

	go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
		sig := <-shutdownSignal
		s.svcLogger.Info("Received shutdown signal.", svc1log.SafeParam("signal", sig.String()))
		if err := s.Shutdown(ctx); err != nil {
			s.svcLogger.Warn("Failed to gracefully shutdown server.", svc1log.Stacktrace(err))
		}
	})
}

// Running returns true if the server is in the "running" state (as opposed to "idle" or "initializing"), false
// otherwise.
func (s *Server[I, R]) Running() bool {
	return s.stateManager.Running()
}

// State returns the state of the current server (idle, initializing or running).
func (s *Server[I, R]) State() ServerState {
	return s.stateManager.State()
}

func (s *Server[I, R]) Shutdown(ctx context.Context) error {
	s.svcLogger.Info("Shutting down server")
	return stopServer(s, func(svr *http.Server) error {
		if err := svr.Shutdown(ctx); err != nil && (ctx.Err() == nil || !errors.Is(err, ctx.Err())) {
			// error is non-nil and not a context error: indicates that there was an error shutting down
			return err
		}
		// errors was nil or server shutdown was interrupted by context completion: either one is considered a successful shutdown
		return nil
	})
}

func (s *Server[I, R]) Close() error {
	s.svcLogger.Info("Closing server")
	return stopServer(s, func(svr *http.Server) error {
		return svr.Close()
	})
}

// decryptConfigBytes returns a version of the provided input bytes in which any values encrypted using the encrypted
// configuration value library are decrypted. If the input bytes do not contain any encrypted configuration values, this
// function is a noop and returns the provided bytes. Otherwise, the provided bytes are interpreted as YAML and any
// encrypted configuration values are decrypted and the resulting bytes are returned.
//
// NOTE: as described in the function comment, if the provided bytes contain any encrypted configuration values, the
// bytes are assumed to be YAML and are treated as such.
func (s *Server[I, R]) decryptConfigBytes(cfgBytes []byte) ([]byte, error) {
	if !encryptedconfigvalue.ContainsEncryptedConfigValueStringVars(cfgBytes) {
		// Nothing to do
		return cfgBytes, nil
	}
	if s.ecvKeyProvider == nil {
		return cfgBytes, werror.Error("No encryption key provider configured but config contains encrypted values")
	}
	ecvKey, err := s.ecvKeyProvider.Load()
	if err != nil {
		return cfgBytes, err
	}
	if ecvKey == nil {
		return cfgBytes, werror.Error("No encryption key configured but config contains encrypted values")
	}
	decryptedBytes, err := decryptECVYAMLNodes(cfgBytes, ecvKey)
	if err != nil {
		return cfgBytes, werror.Wrap(err, "Failed to decrypt values in YAML that contains encrypted values")
	}
	return decryptedBytes, nil
}

// decryptECVYAMLNodes takes the provided YAML bytes and returns equivalent YAML bytes where any scalar nodes with a
// value that consisted of an encrypted configuration value are replaced with the equivalent value that is decrypted
// using the provided key. Does this by unmarshaling the provided bytes into a yamlv3.Node, updating all of the relevant
// values of the Nodes and then marshaling the updated node as bytes. It would be more efficient to decode the yaml.v3
// Node directly to the destination type instead of marshaling it as bytes again, but the existing API requires
// returning []byte so that callers can perform decryption on their own. Previously, ECV values were decrypted directly
// as raw bytes, but this could result in invalid YAML if multi-line values were encrypted. Decrypting values in YAML
// nodes and then writing the nodes back out ensures that the resulting bytes are always valid YAML.
func decryptECVYAMLNodes(yamlBytes []byte, kwt *encryptedconfigvalue.KeyWithType) ([]byte, error) {
	var yamlDocNode yamlv3.Node
	if err := yamlv3.Unmarshal(yamlBytes, &yamlDocNode); err != nil {
		return nil, werror.Wrap(err, "failed to unmarshal YAML into yaml.v3 node")
	}
	if err := decryptNodeValues(&yamlDocNode, kwt); err != nil {
		return nil, err
	}
	return yamlv3.Marshal(&yamlDocNode)
}

// decryptNodeValues recursively modifies the provided node and all of its content nodes such that any nodes that have
// the kind ScalarNode and have a value that contains an encrypted configuration value are modified such that their
// value is the version of the value that is decrypted using the provided KeyWithType.
func decryptNodeValues(n *yamlv3.Node, kwt *encryptedconfigvalue.KeyWithType) error {
	if n == nil {
		return nil
	}
	if n.Kind == yamlv3.ScalarNode && encryptedconfigvalue.ContainsEncryptedConfigValueStringVars([]byte(n.Value)) {
		decrypted := encryptedconfigvalue.DecryptAllEncryptedValueStringVars([]byte(n.Value), *kwt)
		// The existence of encrypted values after an decryption attempt implies decryption failed.
		if encryptedconfigvalue.ContainsEncryptedConfigValueStringVars(decrypted) {
			return werror.Error("failed to decrypt encrypted-config-value in YAML node")
		}
		n.Value = string(decrypted)
	}
	for _, childNode := range n.Content {
		if err := decryptNodeValues(childNode, kwt); err != nil {
			return err
		}
	}
	return nil
}

func stopServer[I config.BaseInstallConfig, R config.BaseRuntimeConfig](s *Server[I, R], stopper func(s *http.Server) error) error {
	// use compare and swap so that the server can only be stopped once
	curState := s.stateManager.State()
	for {
		if curState == ServerIdle || curState == ServerShuttingDown {
			// already shutting down or stopped
			return nil
		}
		// state could be ServerRunning or ServerInitializing
		if s.stateManager.compareAndSwapState(curState, ServerShuttingDown) {
			break
		}
		curState = s.stateManager.State()
	}
	// Only commit to finishing the shutdown if we won the state swap
	defer func() {
		// can avoid compare and swap here because:
		// - Nothing else is allowed to move the state from ServerShuttingDown to ServerIdle
		// - Only one goroutine can ever be in this block at a time due to the above compare and swap
		s.stateManager.setState(ServerIdle)
		close(s.shutdownFinished)
	}()
	if s.httpServer == nil {
		return nil
	}
	return stopper(s.httpServer)
}

func (s *Server[I, R]) getApplicationTracingOptions(install config.Install) []wtracing.TracerOption {
	return getTracingOptions(s.applicationTraceSampler, install, traceSamplerFromSampleRate(defaultSampleRate), install.Server.Port, install.TraceSampleRate)
}

func (s *Server[I, R]) getManagementTracingOptions(install config.Install) []wtracing.TracerOption {
	return getTracingOptions(s.managementTraceSampler, install, neverSample, install.Server.ManagementPort, install.ManagementTraceSampleRate)
}

func getTracingOptions(configuredSampler wtracing.Sampler, install config.Install, fallbackSampler wtracing.Sampler, port int, sampleRate *float64) []wtracing.TracerOption {
	endpoint := &wtracing.Endpoint{
		ServiceName: install.ProductName,
		Port:        uint16(port),
	}
	if parsedIP := net.ParseIP(install.Server.Address); len(parsedIP) > 0 {
		if parsedIP.To4() != nil {
			endpoint.IPv4 = parsedIP
		} else {
			endpoint.IPv6 = parsedIP
		}
	}
	return []wtracing.TracerOption{
		wtracing.WithLocalEndpoint(endpoint),
		getSamplingTraceOption(configuredSampler, fallbackSampler, sampleRate),
	}
}

func getSamplingTraceOption(configuredSampler wtracing.Sampler, fallbackSampler wtracing.Sampler, sampleRate *float64) wtracing.TracerOption {
	if configuredSampler != nil {
		return wtracing.WithSampler(configuredSampler)
	} else if sampleRate != nil {
		return wtracing.WithSampler(traceSamplerFromSampleRate(*sampleRate))
	}
	return wtracing.WithSampler(fallbackSampler)
}

func traceSamplerFromSampleRate(sampleRate float64) wtracing.Sampler {
	if sampleRate <= 0 {
		return neverSample
	}
	if sampleRate >= 1 {
		return alwaysSample
	}
	boundary := uint64(sampleRate * float64(math.MaxUint64)) // does not overflow because we already checked bounds
	return func(id uint64) bool {
		return id < boundary
	}
}

func neverSample(id uint64) bool { return false }

func alwaysSample(id uint64) bool { return true }
