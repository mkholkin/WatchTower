package repo

import "errors"

// Базовые ошибки репозиториев
var (
	ErrNotFound            = errors.New("entity not found")
	ErrAlreadyExists       = errors.New("entity already exists")
	ErrForeignKeyViolation = errors.New("foreign key constraint violation")
	ErrDB                  = errors.New("database error")
	ErrInternal            = errors.New("internal error")
)
