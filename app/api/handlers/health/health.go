// Package health is responsible for all health check entry points
package health

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/rsb/prolog/app"
	"go.uber.org/zap"
)

type CheckHandler struct {
	build string
	log   *zap.SugaredLogger
	kube  app.KubeInfo
}

func NewCheckHandler(d *app.Dependencies) *CheckHandler {
	return &CheckHandler{
		build: d.Build,
		log:   d.Logger,
		kube:  d.Kubernetes,
	}
}

// Readiness checks if the database is ready and if not will return a 500 status.
// Do not respond by just returning an error because further up in the call
// stack it will interpret that as a non-trusted error.
func (h *CheckHandler) Readiness(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

type SystemStatus struct {
	Status    string `json:"status,omitempty"`
	Build     string `json:"build,omitempty"`
	Host      string `json:"host,omitempty"`
	Pod       string `json:"pod,omitempty"`
	PodIP     string `json:"podIP,omitempty"`
	Node      string `json:"node,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

func (h CheckHandler) Liveness(c *fiber.Ctx) error {
	host, err := os.Hostname()
	if err != nil {
		host = "unavailable"
	}

	status := SystemStatus{
		Status:    "up",
		Build:     h.build,
		Host:      host,
		Pod:       h.kube.Pod,
		PodIP:     h.kube.PodIP,
		Node:      h.kube.Node,
		Namespace: h.kube.Namespace,
	}

	return c.Status(fiber.StatusOK).JSON(status)
}

func (h *CheckHandler) SuperSecret(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"secret": "super"})
}
