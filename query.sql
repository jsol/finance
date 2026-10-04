-- name: ListCategories :many
SELECT id, name FROM categories;

-- name: AddCategory :exec
INSERT INTO categories (id, name) VALUES(?,?);

-- name: RemoveCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: UpdateCategory :exec
UPDATE categories SET name = ? WHERE id = ?;

-- name: GetPeriod :many
SELECT id,period, sum, who, cat FROM items WHERE period = ?;

-- name: AddItem :exec
INSERT INTO items (id, period, sum,  who, cat) VALUES(?,?,?,?,?);

-- name: RemoveItem :exec
DELETE FROM items WHERE id = ?;

-- name: GetItem :one
SELECT * FROM items WHERE id = ?;

-- name: GetPatterns :many
SELECT id, pattern, category FROM patterns;

-- name: AddPattern :exec
INSERT INTO patterns (id, pattern, category) VALUES(?,?,?);

-- name: RemovePattern :exec
DELETE FROM patterns WHERE id = ?;

-- name: GetUsers :many
SELECT name, factor FROM users;

-- name: AddUser :exec
INSERT INTO users (name, factor) VALUES(?,?);

-- name: RemoveUser :exec
DELETE FROM users WHERE name = ?;

-- name: UpdateUser :exec
UPDATE users SET factor = ? WHERE name = ?;
