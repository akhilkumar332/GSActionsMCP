-- name: IncrementTemplateUses :one
UPDATE templates SET uses_count = uses_count + 1 WHERE id = $1 RETURNING uses_count;