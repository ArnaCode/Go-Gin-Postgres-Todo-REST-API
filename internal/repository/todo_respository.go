package repository

import (
	"context"
	"fmt"
	"time"
	"todo_api/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func CreateTodo(pool *pgxpool.Pool, title string, completed bool, userID string) (*models.Todo, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
			INSERT INTO todos (title, completed, user_id)
			VALUES ($1, $2, $3)
			RETURNING id, title, completed, created_at, updated_at, user_id
	`

	var todo models.Todo

	var err error = pool.QueryRow(ctx, query, title, completed, userID).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Completed,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.UserID,
	)

	if err != nil {
		return nil, err
	}

	return &todo, nil
}

func GetAllTodos(pool *pgxpool.Pool, userID string) ([]models.Todo, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		SELECT id, title, completed, created_at, updated_at, user_id
		FROM todos
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var rows, err = pool.Query(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var todos []models.Todo = []models.Todo{}

	for rows.Next() {
		var todo models.Todo

		err = rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Completed,
			&todo.CreatedAt,
			&todo.UpdatedAt,
			&todo.UserID,
		)

		if err != nil {
			return nil, err
		}

		todos = append(todos, todo)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

func GetToDoByID(pool *pgxpool.Pool, id int, userID string) (*models.Todo, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		SELECT id, title, completed, created_at, updated_at, user_id
		FROM todos
		WHERE id = $1 AND user_id = $2
	`

	var todo models.Todo

	var err error = pool.QueryRow(ctx, query, id, userID).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Completed,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.UserID,
	)

	if err != nil {
		return nil, err
	}

	return &todo, nil
}

func UpdateToDo(pool *pgxpool.Pool, id int, title string, completed bool, userID string) (*models.Todo, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		UPDATE todos
		SET title = $1, completed = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND user_id = $4
		RETURNING id, title, completed, created_at, updated_at, user_id
	`

	var todo models.Todo

	var err error = pool.QueryRow(ctx, query, title, completed, id, userID).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Completed,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.UserID,
	)

	if err != nil {
		return nil, err
	}

	return &todo, nil
}

func DeleteToDo(pool *pgxpool.Pool, id int, userID string) error {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		DELETE FROM todos
		WHERE id = $1 AND user_id = $2
	`

	var commandTag, err = pool.Exec(ctx, query, id, userID)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("todo with id %d not found", id)
	}

	return nil
}

func GetTodoCompletionFlags(pool *pgxpool.Pool, userID string) ([]bool, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		SELECT completed
		FROM todos
		WHERE user_id = $1
	`

	var rows, err = pool.Query(ctx, query, userID)

	if err != nil {
		return nil, &AppError{Code: "QUERY_FAILED", Message: err.Error()}
	}

	defer rows.Close()

	var flags []bool = []bool{}

	for rows.Next() {
		var completed bool

		err = rows.Scan(&completed)

		if err != nil {
			return nil, &AppError{Code: "SCAN_FAILED", Message: err.Error()}
		}

		flags = append(flags, completed)
	}

	if err = rows.Err(); err != nil {
		return nil, &AppError{Code: "ROWS_FAILED", Message: err.Error()}
	}

	return flags, nil
}
