// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package datastore

type Payment struct {
	RequestedAt   string
	CorrelationID string
	Processor     string
	Amount        float64
}
