-- +goose Up
ALTER TABLE users
ADD COLUMN hashed_password TEXT NOT NULL DEFAULT 'unset';

-- +goose Down
ALTER TABLE users DROP COLUMN hashed_password;

-- goose commandline sa spremljenim podacima je u Postgres.odt-u.
