-- name: GetActiveContentBlock :one
SELECT * FROM content_blocks 
WHERE section_key = $1 
  AND is_active = true 
  AND (start_at IS NULL OR start_at <= NOW())
  AND (end_at IS NULL OR end_at > NOW());

-- name: UpdateContentBlockSchedule :exec
UPDATE content_blocks 
SET start_at = $2, end_at = $3, is_active = $4
WHERE section_key = $1;

-- name: GetContentBlockByKey :one
SELECT * FROM content_blocks WHERE section_key = $1;

-- name: UpsertContentBlock :one
INSERT INTO content_blocks (section_key, content)
VALUES ($1, $2)
ON CONFLICT (section_key) DO UPDATE
SET content = EXCLUDED.content,
    updated_at = NOW()
RETURNING *;
