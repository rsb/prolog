package construct

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rsb/failure"
	"github.com/rsb/prolog/app"
	"github.com/rsb/prolog/app/api/handlers/consume"
	"github.com/rsb/prolog/app/api/handlers/health"
	"github.com/rsb/prolog/app/api/handlers/produce"
	"github.com/rsb/prolog/business"
)

func AddAllRoutes(r *fiber.App, d *app.Dependencies) (*fiber.App, error) {
	r = AddHealthCheckRoutes(r, d)

	l := business.NewLog()
	producer, err := produce.NewHandler(l)
	if err != nil {
		return nil, failure.Wrap(err, "produce.NewHandler failed")
	}

	consumer, err := consume.NewHandler(l)
	if err != nil {
		return nil, failure.Wrap(err, "consume.NewHandler failed")
	}
	r.Post("/", producer.Produce)
	r.Get("/", consumer.Consume)
	return r, nil
}

func AddHealthCheckRoutes(r *fiber.App, d *app.Dependencies) *fiber.App {
	checker := health.NewCheckHandler(d)
	r.Get("/readiness", checker.Readiness)

	return r
}
