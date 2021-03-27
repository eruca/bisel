
-- +migrate Up
CREATE TABLE person(int id);
-- +migrate Down
DROP TABLE IF EXISTS person;