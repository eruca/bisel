package types

type Service func(db *DB, cacher Cacher)
