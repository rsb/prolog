package server

import (
	"context"

	"google.golang.org/grpc"

	"github.com/rsb/failure"
	data "github.com/rsb/prolog/app/api/handlers/v1"
)

type CommitLog interface {
	Append(record *data.Record) (uint64, error)
	Read(offset uint64) (*data.Record, error)
}

type Config struct {
	CommitLog CommitLog
}

var _ data.LogServer = (*GRPCServer)(nil)

type GRPCServer struct {
	data.UnimplementedLogServer
	*Config
}

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newGRPCServer(config)
	if err != nil {
		return nil, failure.Wrap(err, "newGRPCServer failed")
	}

	data.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}
func newGRPCServer(config *Config) (*GRPCServer, error) {
	srv := GRPCServer{Config: config}

	return &srv, nil
}

func (s *GRPCServer) Produce(ctx context.Context, req *data.ProduceRequest) (*data.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, failure.Wrap(err, "s.CommitLog.Append failed")
	}

	return &data.ProduceResponse{Offset: offset}, nil
}

func (s *GRPCServer) ProduceStream(stream data.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return failure.Wrap(err, "stream.Recv failed")
		}

		result, err := s.Produce(stream.Context(), req)
		if err != nil {
			return failure.Wrap(err, "s.Produce failed")
		}

		if err = stream.Send(result); err != nil {
			return failure.Wrap(err, "stream.Send failed")
		}
	}
}

func (s *GRPCServer) Consume(ctx context.Context, req *data.ConsumeRequest) (*data.ConsumeResponse, error) {

	rec, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, failure.Wrap(err, "s.CommitLog.Read failed (%s)", req.Offset)
	}

	return &data.ConsumeResponse{Record: rec}, nil
}

func (s *GRPCServer) ConsumeStream(
	req *data.ConsumeRequest,
	stream data.Log_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			result, err := s.Consume(stream.Context(), req)
			switch {
			case err == nil:
			case failure.IsOutOfRange(err):
				continue
			default:
				return err
			}

			if err = stream.Send(result); err != nil {
				return err
			}

			req.Offset++
		}
	}
}
