package handlers

import (
	"net/http"
	"strconv"
	"todo_api/internal/models"
	"todo_api/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateTodoInput struct {
	Title     string `json:"title" binding:"required"`
	Completed bool   `json:"completed"`
}

type UpdateTodoInput struct {
	Title *string `json:"title"`
	// &true ---------------> set completed as -> true
	// &false ---------------> set completed as -> false
	// nil ---------------> set completed as -> not provided
	Completed *bool `json:"completed"`
}

func CreateTodoHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}

		userID := userIDInterface.(string)

		var input CreateTodoInput

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		todo, err := repository.CreateTodo(pool, input.Title, input.Completed, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		c.JSON(http.StatusCreated, todo)
	}
}

func GetAllTodosHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}
		// interface{} or any{},
		userID := userIDInterface.(string)

		todos, err := repository.GetAllTodos(pool, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, todos)
	}
}

func GetToDoByIDHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}
		// interface{} or any{},
		userID := userIDInterface.(string)

		idStr := c.Param("id")
		// "2" ------------> 2, nil
		// "a" ------------> 0, error ("invalid syntax")

		id, err := strconv.Atoi(idStr)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
			return
		}

		todo, err := repository.GetToDoByID(pool, id, userID)

		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		c.JSON(http.StatusOK, todo)
	}
}

func UpdateToDoHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}
		// interface{} or any{},
		userID := userIDInterface.(string)

		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})

			return
		}

		var input UpdateTodoInput

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if input.Title == nil && input.Completed == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field (title or completed) must be provided"})
			return
		}

		existing, err := repository.GetToDoByID(pool, id, userID)

		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		title := existing.Title
		if input.Title != nil {
			title = *input.Title
		}

		completed := existing.Completed
		if input.Completed != nil {
			completed = *input.Completed
		}

		todo, err := repository.UpdateToDo(pool, id, title, completed, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, todo)

	}
}

func DeleteToDoHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}
		// interface{} or any{},
		userID := userIDInterface.(string)

		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})

			return
		}

		err = repository.DeleteToDo(pool, id, userID)

		if err != nil {
			if err.Error() == "todo with id "+idStr+" not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
	}
}

func fetchCompletionFlags(pool *pgxpool.Pool, userID string) ([]bool, error) {
	return repository.GetTodoCompletionFlags(pool, userID)
}

func GetTodoStatsHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}

		userID := userIDInterface.(string)

		flags, err := fetchCompletionFlags(pool, userID)

		// fmt.Printf("type: %T | value: %v | err == nil: %v\n", err, err, err == nil)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load todo statistics"})
			return
		}

		stats := make(map[string]int)

		stats["completed"] = 0
		stats["pending"] = 0

		for _, completed := range flags {
			if completed {
				stats["completed"]++
			} else {
				stats["pending"]++
			}
		}

		c.JSON(http.StatusOK, stats)
	}
}

// [EP03 · Slice Aliasing & the Append Bug] ✅ NEW
// 🐞 BUG: result is built on todos[:0], so it shares the backing array of
// todos, and every append overwrites the caller's todos.

// filterTodos returns the todos in todos whose Completed field equals
// completed, in their original order.
func filterTodos(todos []models.Todo, completed bool) []models.Todo {
	result := todos[:0]

	for _, todo := range todos {
		if todo.Completed == completed {
			result = append(result, todo)
		}
	}

	return result
}

// [EP03 · Slice Aliasing & the Append Bug] ✅ NEW

// GetGroupedTodosHandler returns a handler for GET /todos/grouped. It
// responds with the authenticated user's todos split into two lists, each
// newest first:
//
//	{"completed": [...], "pending": [...]}
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. If the todos cannot be loaded, the handler responds with 500.
func GetGroupedTodosHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not found in context"})
			return
		}

		userID := userIDInterface.(string)

		todos, err := repository.GetAllTodos(pool, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load todos"})
			return
		}

		completed := filterTodos(todos, true)
		pending := filterTodos(todos, false)

		c.JSON(http.StatusOK, gin.H{
			"completed": completed,
			"pending":   pending,
		})
	}
}
