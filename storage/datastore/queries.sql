-- name: LogPayment :exec
INSERT INTO Payment (requested_at, correlation_id, processor, amount)
             VALUES (           ?,              ?,         ?,      ?);

-- name: ExpungePayment :exec
DELETE FROM Payment WHERE correlation_id = ?;
