module github.com/palantir/witchcraft-go-server/v2

go 1.16

require (
	github.com/gorilla/mux v1.7.3
	github.com/julienschmidt/httprouter v1.3.0
	github.com/nmiyake/pkg/dirs v1.0.0
	github.com/palantir/conjure-go-runtime/v2 v2.26.0
	github.com/palantir/go-encrypted-config-value v1.1.0
	github.com/palantir/go-metrics v1.1.1
	github.com/palantir/pkg/httpserver v1.0.1
	github.com/palantir/pkg/metrics v1.3.0
	github.com/palantir/pkg/objmatcher v1.0.1
	github.com/palantir/pkg/refreshable v1.3.2
	github.com/palantir/pkg/safejson v1.0.1
	github.com/palantir/pkg/signals v1.0.1
	github.com/palantir/pkg/tlsconfig v1.1.0
	github.com/palantir/witchcraft-go-error v1.5.0
	github.com/palantir/witchcraft-go-health v1.11.0
	github.com/palantir/witchcraft-go-logging v1.17.0
	github.com/palantir/witchcraft-go-params v1.2.0
	github.com/palantir/witchcraft-go-tracing v1.4.0
	github.com/stretchr/testify v1.7.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)
