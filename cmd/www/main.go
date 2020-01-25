package main

import (
	"github.com/deissh/osu-api-server/pkg"
	"github.com/deissh/osu-api-server/pkg/middlewares/customerror"
	"github.com/deissh/osu-api-server/pkg/middlewares/customlogger"
	"github.com/deissh/osu-api-server/pkg/oauth"
	"github.com/deissh/osu-api-server/pkg/v2"
	"github.com/getsentry/sentry-go"
	sentryEcho "github.com/getsentry/sentry-go/echo"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"context"
	"os"
	"os/signal"
	"time"
)

func main() {
	// loading configuration
	config.WithOptions(config.ParseEnv, config.Readonly)
	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("config.yaml")
	if err != nil {
		panic(err)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if config.Bool("debug", false) {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(
			zerolog.ConsoleWriter{
				Out:     os.Stderr,
				NoColor: false,
			},
		).With().Caller().Logger()
	}

	log.Debug().
		Msg("Loaded configuration and logger")

	log.Debug().
		Msg("Start initialize database and redis")

	pkg.InitializeDB()
	pkg.InitializeRedis()

	log.Debug().
		Msg("Initialize database and redis successful done")

	// Seting up Echo
	app := echo.New()
	app.HideBanner = true
	app.HTTPErrorHandler = customerror.CustomHTTPErrorHandler

	log.Debug().
		Msg("Setting up Echo middleware")

	app.Use(middleware.RequestID())
	// app.Use(middleware.Recover())
	app.Use(customlogger.Middleware())

	if config.Bool("server.cors.enable") {
		log.Info().
			Msg("Enabled build-in CORS")

		app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: config.Strings("server.cors.allow_origins"),
			AllowHeaders: config.Strings("server.cors.allow_headers"),
		}))
	}

	if config.Bool("server.sentry.enable") {
		log.Debug().
			Msg("Start initialize Sentry")

		err = sentry.Init(sentry.ClientOptions{
			Dsn: config.String("server.sentry.dsn"),
		})
		if err != nil {
			log.Error().Err(err).Msg("Sentry initialization failed")
		}

		app.Use(sentryEcho.New(sentryEcho.Options{}))

		log.Debug().
			Msg("Initialize Sentry successful done")
	}

	log.Debug().
		Msg("Mounting Echo routes")

	oauth.ApplyRoutes(app.Group(""))
	v2.ApplyRoutes(app.Group("/api"))

	log.Debug().
		Msg("Running HTTP server")

	// Graceful start and stop HTTP server
	go func() {
		err := app.Start(config.String("server.host") + ":" + config.String("server.port"))
		if err != nil {
			log.Error().
				Err(err).
				Msg("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Fatal().
			Err(err).
			Send()
	}
}
