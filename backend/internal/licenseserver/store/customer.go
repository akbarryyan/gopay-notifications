package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrCustomerNotFound = errors.New("store: customer tidak ditemukan")

type Customer struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

func NewCustomerID() (string, error) { return randomPrefixedID("cus_") }

func (s *Store) CreateCustomer(ctx context.Context, id, name string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO customers (id, name) VALUES ($1, $2)`, id, name)
	if err != nil {
		return fmt.Errorf("store: create customer: %w", err)
	}
	return nil
}

// ListCustomers mengembalikan seluruh customer, terbaru lebih dulu.
func (s *Store) ListCustomers(ctx context.Context) ([]Customer, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, created_at FROM customers ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list customers: %w", err)
	}
	defer rows.Close()

	var out []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan customer: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCustomer(ctx context.Context, id string) (Customer, error) {
	var c Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, created_at FROM customers WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrCustomerNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("store: get customer: %w", err)
	}
	return c, nil
}
