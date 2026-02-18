package database

import (
	"context"
	"data-explorer/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

func DefaultConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     5432,
		User:     "indexer",
		Password: "indexer_password",
		DBName:   "blockchain_explorer",
		SSLMode:  "disable",
		MaxConns: 25,
		MinConns: 5,
	}
}

type DB struct {
	conn *sql.DB
}

func NewDB(cfg *Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(cfg.MaxConns)
	conn.SetMaxIdleConns(cfg.MinConns)
	conn.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to PostgreSQL database: %s", cfg.DBName)
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, opts)
}

func (db *DB) InsertBlock(ctx context.Context, block *utils.Block) error {
	query := `
		INSERT INTO blocks (num, hash, parent_hash, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (num) DO NOTHING
	`
	_, err := db.conn.ExecContext(ctx, query,
		block.Num,
		block.Hash,
		block.ParentHash,
		block.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert block %d: %w", block.Num, err)
	}
	return nil
}

func (db *DB) InsertTransaction(ctx context.Context, blockNum int64, blockHash []byte, tx *utils.DecodedTx) error {
	txParamsJSON, err := json.Marshal(tx.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal tx params: %w", err)
	}

	var valueStr *string
	if tx.Value != nil {
		v := tx.Value.String()
		valueStr = &v
	}

	query := `
		SELECT pg_advisory_xact_lock(hashtext($1::text));
		INSERT INTO actions (
			block_num, block_hash, tx_hash, tx_index,
			from_addr, to_addr, contract,
			method, tx_params, value, events
		)
		VALUES ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE((SELECT events FROM actions WHERE block_num = $2 AND tx_hash = $4), '[]'::jsonb))
		ON CONFLICT (block_num, tx_hash) DO UPDATE SET
			method = EXCLUDED.method,
			tx_params = EXCLUDED.tx_params,
			from_addr = EXCLUDED.from_addr,
			to_addr = EXCLUDED.to_addr,
			contract = EXCLUDED.contract,
			value = EXCLUDED.value
	`

	_, err = db.conn.ExecContext(ctx, query,
		tx.TxHash.Hex(),
		blockNum,
		blockHash,
		tx.TxHash.Bytes(),
		tx.TxIndex,
		tx.From.Bytes(),
		tx.To.Bytes(),
		tx.To.Bytes(),
		tx.MethodName,
		txParamsJSON,
		valueStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}
	return nil
}

func (db *DB) UpdateEventsForTransaction(ctx context.Context, blockNum int64, txHash []byte, events []*utils.DecodedEvent) error {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	query := `
		SELECT pg_advisory_xact_lock(hashtext($1::text));
		INSERT INTO actions (block_num, block_hash, tx_hash, tx_index, from_addr, to_addr, contract, events)
		VALUES ($2, $3, $4, 0, $4, $4, $4, $5::jsonb)
		ON CONFLICT (block_num, tx_hash) DO UPDATE SET
			events = COALESCE(actions.events, '[]'::jsonb) || EXCLUDED.events
	`

	_, err = db.conn.ExecContext(ctx, query, string(txHash), blockNum, []byte{}, txHash, eventsJSON)
	if err != nil {
		return fmt.Errorf("failed to update events: %w", err)
	}
	return nil
}

func (db *DB) InsertTransactionBatch(ctx context.Context, blockNum int64, blockHash []byte, txs []*utils.DecodedTx) error {
	if len(txs) == 0 {
		return nil
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, decodedTx := range txs {
		txParamsJSON, err := json.Marshal(decodedTx.Params)
		if err != nil {
			return fmt.Errorf("failed to marshal tx params: %w", err)
		}

		var valueStr *string
		if decodedTx.Value != nil {
			v := decodedTx.Value.String()
			valueStr = &v
		}

		query := `
			SELECT pg_advisory_xact_lock(hashtext($1::text));
			INSERT INTO actions (
				block_num, block_hash, tx_hash, tx_index,
				from_addr, to_addr, contract,
				method, tx_params, value, events
			)
			VALUES ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE((SELECT events FROM actions WHERE block_num = $2 AND tx_hash = $4), '[]'::jsonb))
			ON CONFLICT (block_num, tx_hash) DO UPDATE SET
				method = EXCLUDED.method,
				tx_params = EXCLUDED.tx_params,
				from_addr = EXCLUDED.from_addr,
				to_addr = EXCLUDED.to_addr,
				contract = EXCLUDED.contract,
				value = EXCLUDED.value
		`

		_, err = tx.ExecContext(ctx, query,
			decodedTx.TxHash.Hex(),
			blockNum,
			blockHash,
			decodedTx.TxHash.Bytes(),
			decodedTx.TxIndex,
			decodedTx.From.Bytes(),
			decodedTx.To.Bytes(),
			decodedTx.To.Bytes(),
			decodedTx.MethodName,
			txParamsJSON,
			valueStr,
		)
		if err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Inserted %d transactions", len(txs))
	return nil
}

func (db *DB) UpdateEventsBatch(ctx context.Context, blockNum int64, eventsByTxHash map[string][]*utils.DecodedEvent) error {
	if len(eventsByTxHash) == 0 {
		return nil
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	for txHashStr, events := range eventsByTxHash {
		eventsJSON, err := json.Marshal(events)
		if err != nil {
			return fmt.Errorf("failed to marshal events: %w", err)
		}

		query := `
			SELECT pg_advisory_xact_lock(hashtext($1::text));
			INSERT INTO actions (block_num, block_hash, tx_hash, tx_index, from_addr, to_addr, contract, events)
			VALUES ($2, $3, $4, 0, $4, $4, $4, $5::jsonb)
			ON CONFLICT (block_num, tx_hash) DO UPDATE SET
				events = COALESCE(actions.events, '[]'::jsonb) || EXCLUDED.events
		`

		_, err = tx.ExecContext(ctx, query, txHashStr, blockNum, []byte{}, []byte(txHashStr), string(eventsJSON))
		_, err = stmt.ExecContext(ctx, string(eventsJSON), blockNum, []byte(txHashStr))
		if err != nil {
			return fmt.Errorf("failed to update events: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Updated events for %d transactions", len(eventsByTxHash))
	return nil
}
