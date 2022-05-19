package produce

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rsb/failure"
	"github.com/rsb/prolog/business"
)

type Request struct {
	Record business.Record `json:"record"`
}

type Response struct {
	Offset uint64 `json:"offset"`
}

type Handler struct {
	log *business.Log
}

func NewHandler(l *business.Log) (*Handler, error) {
	if l == nil {
		return nil, failure.InvalidParam("[l] business.Log is nil")
	}

	return &Handler{log: l}, nil
}

func (h *Handler) Produce(c *fiber.Ctx) error {
	var req Request

	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return failure.ToBadRequest(err, "invalid request")
	}
	fmt.Printf("%+v\n", req.Record)
	off, err := h.log.Append(req.Record)
	if err != nil {
		return failure.ToSystem(err, "h.log.Append failed")
	}

	resp := Response{Offset: off}
	return c.Status(http.StatusOK).JSON(&resp)
}
