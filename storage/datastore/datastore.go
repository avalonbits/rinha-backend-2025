package datastore

import (
	"embed"

	"github.com/avalonbits/rinha-backend-2025/storage"
)

type DB = storage.DB[Queries]

//go:embed migrations/*
var Migrations embed.FS

func Factory(tx storage.DBTX) *Queries {
	return New(tx)
}
