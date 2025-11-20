-- name: CreateToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES(
    $1,
    Now(),
    Now(),
    $2,
    $3
)
RETURNING * ;


-- name: GetUserFromRefreshToken :one 
SELECT u.*
FROM refresh_tokens r
INNER JOIN users u ON r.user_id = u.id 
WHERE r.token = $1 AND r.revoked_at IS NULL;


-- name: GetRefreshTokenByToken :one
SELECT * FROM refresh_tokens WHERE token = $1 and revoked_at IS NULL ;

-- name: RevokeToken :exec
UPDATE refresh_tokens 
SET revoked_at = Now(), updated_at = NOW()
WHERE token = $1;


