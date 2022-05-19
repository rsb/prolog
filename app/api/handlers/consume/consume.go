package consume

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rsb/failure"
	"github.com/rsb/prolog/business"
)

type Handler struct {
	log *business.Log
}

type Request struct {
	Offset uint64 `json:"offset"`
}

type Response struct {
	Record business.Record `json:"record"`
}

func NewHandler(l *business.Log) (*Handler, error) {
	if l == nil {
		return nil, failure.InvalidParam("[l] business.Log is nil")
	}

	return &Handler{log: l}, nil
}

func (h *Handler) Consume(c *fiber.Ctx) error {
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return failure.ToBadRequest(err, "invalid request")
	}

	rec, err := h.log.Read(req.Offset)
	if err != nil {
		return failure.ToSystem(err, "h.log.Append failed")
	}

	resp := Response{Record: rec}
	return c.Status(http.StatusOK).JSON(&resp)
}
