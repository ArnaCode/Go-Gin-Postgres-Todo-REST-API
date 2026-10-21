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

// [EP01 · Missing Return After c.JSON]
// 🐞 BUG: this handler writes a response on the repository error path and then
// continues. c.JSON does not end the handler.

// CreateTodoHandler returns a handler for POST /todos. It creates a todo
// owned by the authenticated user from the title and completed flag in the
// request body, and responds with 201 and the created todo.
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. The handler responds with 400 if the body is invalid or has no title,
// and 500 if the insert fails.
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

		// 🐞 BUG 4: on an insert failure this sends a 500 and then falls through
		// to the 201 below with todo == nil, appending "null" to the body.
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			// ✅ NEW CODE
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

// [EP01 · Missing Return After c.JSON]
// 🐞 BUG: the catch-all error branch in this handler does not stop execution.
// Compare it with the ErrNoRows branch directly above it, which does.

// GetToDoByIDHandler returns a handler for GET /todos/:id. It responds with
// the authenticated user's todo with the given ID.
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. The handler responds with 400 if the ID is not a number, 404 if the
// user has no todo with that ID, and 500 if the lookup fails.
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

			// 🐞 BUG 5: any error that is not ErrNoRows sends a 500 here and then
			// reaches the 200 below with todo == nil.
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			// ✅ NEW CODE
			return
		}

		c.JSON(http.StatusOK, todo)
	}
}

// [EP01 · Missing Return After c.JSON]
// 🐞 BUG: the ID validation branch here rejects the input and then keeps going.
// The same branch in GetToDoByIDHandler stops correctly.

// UpdateToDoHandler returns a handler for PUT /todos/:id. It changes the
// title, the completed flag, or both, of the authenticated user's todo with
// the given ID, and responds with the updated todo. A field left out of the
// request body keeps its current value.
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. The handler responds with 400 if the ID is not a number or the body is
// invalid or changes nothing, 404 if the user has no todo with that ID, and
// 500 if the update fails.
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

		// 🐞 BUG 3: a non-numeric ID sends a 400 and then continues with id == 0,
		// so the body is still bound and a lookup still runs against the
		// database.
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
			// ✅ NEW CODE
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

// [EP01 · Missing Return After c.JSON]
// 🐞 BUG: two branches in this handler write a response without returning, so a
// single request can produce three JSON objects and still run a DELETE query.

// DeleteToDoHandler returns a handler for DELETE /todos/:id. It deletes the
// authenticated user's todo with the given ID and responds with a
// confirmation message.
//
// The user ID is read from the request context, where AuthMiddleware stores
// it. The handler responds with 400 if the ID is not a number, 404 if the
// user has no todo with that ID, and 500 if the deletion fails.
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

		// 🐞 BUG 1: a non-numeric ID sends a 400 and then continues with id == 0,
		// so the DELETE below executes against the database anyway.
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
			// ✅ NEW CODE
			return
		}

		err = repository.DeleteToDo(pool, id, userID)

		if err != nil {
			if err.Error() == "todo with id "+idStr+" not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
				return
			}

			// 🐞 BUG 2: sends a second body and then falls through to the success
			// message below, so the response reports a deletion that never
			// happened.
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			// ✅ NEW CODE
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
	}
}
