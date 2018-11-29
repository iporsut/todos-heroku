package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Server struct {
	db *sql.DB
}

type Todo struct {
	ID        int64     `json:"id"`
	Body      string    `json:"todo"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Server) All(c *gin.Context) {
	rows, err := s.db.Query("SELECT id, todo, updated_at, created_at FROM todos")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("db: query error: %s", err),
		})
		return
	}
	todos := []Todo{} // set empty slice without nil
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Body, &todo.UpdatedAt, &todo.CreatedAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"object":  "error",
				"message": fmt.Sprintf("db: query error: %s", err),
			})
			return
		}
		todos = append(todos, todo)
	}
	c.JSON(http.StatusOK, todos)
}

func (s *Server) Create(c *gin.Context) {
	var todo Todo
	err := c.ShouldBindJSON(&todo)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	row := s.db.QueryRow("INSERT INTO todos (todo, created_at, updated_at) values ($1, $2, $3) RETURNING id", todo.Body, now, now)

	if err := row.Scan(&todo.ID); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("db: query error: %s", err),
		})
		return
	}

	c.JSON(http.StatusCreated, todo)
}

func (s *Server) GetByID(c *gin.Context) {
	stmt := "SELECT id, todo, created_at, updated_at FROM todos WHERE id = $1"
	id, _ := strconv.Atoi(c.Param("id"))
	row := s.db.QueryRow(stmt, id)
	var todo Todo
	err := row.Scan(&todo.ID, &todo.Body, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, todo)
}

func (s *Server) DeleteByID(c *gin.Context) {
	stmt := "DELETE FROM todos WHERE id = $1"
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := s.db.Exec(stmt, id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) Update(c *gin.Context) {
	h := gin.H{}
	if err := c.ShouldBindJSON(&h); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	stmt := "UPDATE todos SET todo = $2 WHERE id = $1"
	_, err := s.db.Exec(stmt, id, h["todo"])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	stmt = "SELECT id, todo, created_at, updated_at FROM todos WHERE id = $1"
	row := s.db.QueryRow(stmt, id)
	var todo Todo
	err = row.Scan(&todo.ID, &todo.Body, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, todo)
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	createTable := `
	CREATE TABLE IF NOT EXISTS todos (
		id SERIAL PRIMARY KEY,
		todo TEXT,
		created_at TIMESTAMP WITHOUT TIME ZONE,
		updated_at TIMESTAMP WITHOUT TIME ZONE
	);
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}

	s := &Server{
		db: db,
	}
	r := gin.Default()
	r.GET("/todos", s.All)
	r.POST("/todos", s.Create)

	r.GET("/todos/:id", s.GetByID)
	r.PUT("/todos/:id", s.Update)
	r.DELETE("/todos/:id", s.DeleteByID)

	r.Run(":" + os.Getenv("PORT"))
}
