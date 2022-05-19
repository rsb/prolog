package construct

import (
	"os"
	"time"

	"github.com/rsb/prolog/app/conf"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	expvarmw "github.com/gofiber/fiber/v2/middleware/expvar"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/rsb/failure"
	"github.com/rsb/prolog/app"
	"github.com/rsb/prolog/foundation/logging"
	"go.uber.org/zap"

	"github.com/rsb/prolog/app/api/handlers/health"
)

const (
	DefaultHTTPClientTimeout             = 5 * time.Second
	DefaultHTTPClientMaxIde              = 100
	DefaultHTTPClientMaxConnsPerHost     = 100
	DefaultHTTPClientMaxIdleConnsPerHost = 100
)

func NewLogger(appVersion string) (*zap.SugaredLogger, error) {
	l, err := logging.NewLogger(app.ServiceName, appVersion)
	if err != nil {
		return nil, failure.Wrap(err, "logging.NewLogger failed")
	}

	return l, nil
}

func NewAPIDependencies(sd chan os.Signal, l *zap.SugaredLogger, c conf.PrologAPI) (app.Dependencies, error) {
	var d app.Dependencies
	if sd == nil {
		return d, failure.InvalidParam("sd(chan os.Signal) is nil")
	}

	if l == nil {
		return d, failure.InvalidParam("l(*zap.SugaredLogger) is nil")
	}

	build := c.Version.Build
	if build == "" {
		build = "unavailable"
	}

	d = app.Dependencies{
		Build:           build,
		Host:            c.API.Host,
		DebugHost:       c.API.DebugHost,
		ReadTimout:      c.API.ReadTimeout,
		WriteTimeout:    c.API.WriteTimeout,
		IdleTimeout:     c.API.IdleTimeout,
		ShutdownTimeout: c.API.ShutdownTimeout,
		Shutdown:        sd,
		Logger:          l,
		Kubernetes: app.KubeInfo{
			Pod:       c.Kubernetes.Pod,
			PodIP:     c.Kubernetes.PodIP,
			Node:      c.Kubernetes.Node,
			Namespace: c.Kubernetes.Namespace,
		},
	}

	return d, nil
}

// NewDebugMux registers all the debug standard library routes and then custom
// debug application routes for the service. This bypassing the use of the
// DefaultServerMux. Using the DefaultServerMux would be a security risk since
// a dependency could inject a handler into our service without us knowing it.
func NewDebugMux(d *app.Dependencies) *fiber.App {
	r := fiber.New()
	r.Use(pprof.New())
	r.Use(expvarmw.New())
	h := health.NewCheckHandler(d)

	r.Get("/debug/readiness", h.Readiness)
	r.Get("/debug/liveness", h.Liveness)

	return r
}

func NewAPIMux(d app.Dependencies, c conf.API) *fiber.App {

	app := fiber.New(c.NewFiberConfig())
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(fiberzap.New(
		fiberzap.Config{
			Logger: d.Logger.Desugar(),
		},
	))

	return app
}
