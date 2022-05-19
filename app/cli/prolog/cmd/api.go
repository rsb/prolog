package cmd

import (
	"context"
	"expvar"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/rsb/failure"
	"github.com/rsb/prolog/app"
	"github.com/rsb/prolog/app/conf"
	"github.com/rsb/prolog/app/construct"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

func init() {
	var b conf.PrologAPI
	bindCLI(apiCmd, viper.GetViper(), &b)
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "controls the prolog service",
	Long: `prolog api can be started and stopped using
serve - start the services server
stop  - shutdown the services server
`,
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the lola-api web server",
	Long:  `serve will start an instance of fiber to server the lola-api`,
	RunE:  serveAPI,
}

func serveAPI(_ *cobra.Command, _ []string) error {
	log, err := construct.NewLogger(app.ServiceName)
	if err != nil {
		return failure.Wrap(err, "construct.NewLogger failed (%s)", app.ServiceName)
	}
	defer func() { _ = log.Sync() }()

	var c conf.PrologAPI
	if err = processConfigCLI(viper.GetViper(), &c); err != nil {
		failureExit(log, err, "startup", "processConfigCLI failed")
	}

	c.Version.Build = build

	if err = runAPI(c, log); err != nil {
		failureExit(log, err, "startup", "runAPI failed")
	}

	return nil
}

func runAPI(config conf.PrologAPI, log *zap.SugaredLogger) error {
	// Set the correct number of threads for the services
	// based on what is available either by the machine or quotes
	opt := maxprocs.Logger(log.Infof)
	if _, err := maxprocs.Set(opt); err != nil {
		return failure.ToSystem(err, "maxprocs.Set failed")
	}

	log.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))
	logConfig(log, "startup", config)

	expvar.NewString("build").Set(build)
	ctx := context.Background()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	depend, err := construct.NewAPIDependencies(shutdown, log, config)
	if err != nil {
		return failure.Wrap(err, "construct.NewAPIDependencies failed")
	}

	debugMux := construct.NewDebugMux(&depend)

	// Start the service listening for debug requests.
	// Not concerned with shutting this down with load shedding
	go func() {
		if err = debugMux.Listen(config.API.DebugHost); err != nil {
			log.Errorw("shutdown",
				"status", "debug router closed",
				"host", config.DebugHost,
				"ERROR", err,
			)
		}
	}()

	apiMux := construct.NewAPIMux(depend, config.API)
	apiMux = construct.AddAllRoutes(apiMux, &depend)
	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for api requests
	go func() {
		log.Infow("startup",
			"status", "api router started",
			"host", config.API.Host,
		)
		if err = apiMux.Listen(config.API.Host); err != nil {
			serverErrors <- err
		}
	}()

	// Blocking main and waiting for shutdown
	select {
	case err = <-serverErrors:
		return failure.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Infow("shutdown", "status", "shutdown started", "signal", sig)

		// Give outstanding requests a deadline for completion
		_, cancel := context.WithTimeout(ctx, config.API.ShutdownTimeout)
		defer cancel()

		// Asking listener to shut down and shed load
		if sErr := apiMux.Shutdown(); sErr != nil {
			return failure.Wrap(sErr, "could not stop server gracefully")
		}
	}

	return nil
}

func logConfig(log *zap.SugaredLogger, cat string, c conf.PrologAPI) {
	api := c.API
	log.Infow(cat,
		"version", c.Version.Build,
		"host", api.Host,
		"debug-host", api.DebugHost,
		"read-timeout", api.ReadTimeout,
		"write-timeout", api.WriteTimeout,
		"idle-timeout", api.IdleTimeout,
		"shutdown-timeout", api.ShutdownTimeout,
	)
}
