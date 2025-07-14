-- name: LogPayment :exec
INSERT INTO Payment (requested_at, correlation_id, processor, amount)
             VALUES (           ?,              ?,         ?,      ?);

-- name: ExpungePayment :exec
DELETE FROM Payment WHERE correlation_id = ?;

-- name: PaymentSummary :many
SELECT processor, count(*) total, sum(amount) amount FROM Payment
    WHERE requested_at BETWEEN ? AND ?
GROUP BY processor;
