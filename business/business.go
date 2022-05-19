// Package business is responsible for the business logic and types that
// directly relate to the logging. In the book we are building a logging
// service so in this case we are in the business of logs.
//
// business does not import any sub packages below it. Also, it does not
// import third party dependencies like api's or database adapters, It
// can use package that directly support the business.
//
// The features package is used implement business use cases which can
// import types from this package.
package business

import (
	"sync"

	"github.com/rsb/failure"
)

type Log struct {
	mu      sync.Mutex
	records []Record
}

func NewLog() *Log {
	return &Log{}
}

func (c *Log) Append(rec Record) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rec.Offset = uint64(len(c.records))
	c.records = append(c.records, rec)

	return rec.Offset, nil
}

func (c *Log) Read(offset uint64) (Record, error) {
	var result Record
	c.mu.Lock()
	defer c.mu.Unlock()

	if offset >= uint64(len(c.records)) {
		return result, failure.NotFound("offset")
	}

	return c.records[offset], nil
}

type Record struct {
	Value  []byte `json:"value"`
	Offset uint64 `json:"offset"`
}
