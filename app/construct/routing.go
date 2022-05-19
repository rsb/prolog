package construct

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rsb/prolog/app"
	"github.com/rsb/prolog/app/api/handlers/health"
)

func AddAllRoutes(r *fiber.App, d *app.Dependencies) *fiber.App {
	r = AddHealthCheckRoutes(r, d)
	return r
}

func AddHealthCheckRoutes(r *fiber.App, d *app.Dependencies) *fiber.App {
	checker := health.NewCheckHandler(d)
	r.Get("/readiness", checker.Readiness)
	// a := r.Group("/authorized")

	// a.Use(d.Auth.AuthMiddleware())
	// {
	// 	a.Get("/liviness", checker.Liveness)
	// }

	return r
}
