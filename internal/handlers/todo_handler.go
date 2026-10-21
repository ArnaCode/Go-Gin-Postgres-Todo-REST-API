package handlers

import (
	"net/http"
	"strconv"
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

// [EP02 · Nil Map & Typed-Nil Error Traps] ✅ NEW

// fetchCompletionFlags returns the completed flag of every todo owned by
// userID. It wraps repository.GetTodoCompletionFlags so that handlers
// receive a plain error, like every other repository call in this file.
func fetchCompletionFlags(pool *pgxpool.Pool, userID string) ([]bool, error) {
	return repository.GetTodoCompletionFlags(pool, userID)
}

// [EP02 · Nil Map & Typed-Nil Error Traps] ✅ NEW
// 🐞 BUG 2: stats is declared with var instead of make, so writing to it panics.

// GetTodoStatsHandler returns a handler for GET /todos/stats. It responds
// with the number of completed and pending todos owned by the authenticated
// user:
//
//	{"completed": 1, "pending": 1}
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. If the todos cannot be loaded, the handler responds with 500.
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

		// [EP02 · Nil Map & Typed-Nil Error Traps] ✅ NEW
		// 🐞 BUG 2: stats is declared with var instead of make, so writing to it panics.
		// 🟣 FIX 2: stats is created with make, and both keys start at 0.

		// 🟣 UPDATED — make creates the map's storage
		stats := make(map[string]int)

		// ✅ NEW — both keys always present
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
