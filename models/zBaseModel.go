package models

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorResponse struct {
	Error any `json:"error"`
}

type SuccessResponse struct {
	Notice string `json:"notice,omitempty"`
	Record any    `json:"record,omitempty"`
}

type SuccessResponseWithMeta struct {
	Notice string         `json:"notice,omitempty"`
	Meta   *PaginatorData `json:"_meta,omitempty"`
	Record any            `json:"record,omitempty"`
}

type ConnDb struct {
	ConnPgsql *pgxpool.Pool
	Ctx       context.Context
}

type WorkerStatus struct {
	TaskName    string         `json:"task"`
	PgsqlStatus PoolStatsPgsql `json:"PgsqlConns"`
}

type PoolStatsPgsql struct {
	AcquiredConns   int32  `json:"acquired"`
	TotalConns      int32  `json:"total"`
	IdleConns       int32  `json:"idle"`
	AcquireCount    int64  `json:"acquire_count"`
	AcquireDuration string `json:"acquire_duration"`
	// MaxConns             int32  `json:"max_conns"`
	// CanceledAcquireCount int64  `json:"canceled_acquire_count"`
	// ConstructingConns    int32  `json:"constructing_conns"`
	// EmptyAcquireCount    int64  `json:"empty_acquire_count"`
}

type WorkerStatusMysql struct {
	TaskName    string         `json:"task"`
	PgsqlStatus PoolStatsPgsql `json:"PgsqlConns"`
	MysqlStatus PoolStatsMysql `json:"MysqlConns"`
}

type ConnMysqlPgsql struct {
	ConnMysql *sql.DB
	ConnPgsql *pgxpool.Pool
	Ctx       context.Context
}

type PoolStatsMysql struct {
	MaxOpenConnections int    `json:"max_open"`
	OpenConnections    int    `json:"open"`
	InUse              int    `json:"in_use"`
	Idle               int    `json:"idle"`
	WaitCount          int64  `json:"wait_count"`
	WaitDuration       string `json:"wait_duration"`
	MaxIdleClosed      int64  `json:"max_idle_closed"`
	MaxLifetimeClosed  int64  `json:"max_lifetime_closed"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// datos besser
type FacturaDatosBesser struct {
	Docid     string `json:"docid"`
	Direccion string `json:"direccion"`
	Telefono  string `json:"telefono"`
	Email     string `json:"email"`
}

// Constructor with default values
func GetDatosBesser() FacturaDatosBesser {
	return FacturaDatosBesser{
		Docid:     "j400697506",
		Direccion: "san miguel edif. asdrubal jose, piso pb, planta baja, urb. santa irene, punto fijo, falcòn",
		Telefono:  "02693910500",
		Email:     "atcliente@bessersolutions.com",
	}
}
