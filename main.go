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

type TodoService interface {
	All() ([]Todo, error)
	Insert(todo *Todo) error
	GetByID(id int) (*Todo, error)
	DeleteByID(id int) error
	Update(id int, body string) (*Todo, error)
}

type TodoServiceImp struct {
	db *sql.DB
}

func (s *TodoServiceImp) All() ([]Todo, error) {
	rows, err := s.db.Query("SELECT id, todo, updated_at, created_at FROM todos")
	if err != nil {
		return nil, err
	}
	todos := []Todo{} // set empty slice without nil
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Body, &todo.UpdatedAt, &todo.CreatedAt)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func (s *TodoServiceImp) Insert(todo *Todo) error {
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	row := s.db.QueryRow("INSERT INTO todos (todo, created_at, updated_at) values ($1, $2, $3) RETURNING id", todo.Body, now, now)

	if err := row.Scan(&todo.ID); err != nil {
		return err
	}
	return nil
}

func (s *TodoServiceImp) GetByID(id int) (*Todo, error) {
	stmt := "SELECT id, todo, created_at, updated_at FROM todos WHERE id = $1"
	row := s.db.QueryRow(stmt, id)
	var todo Todo
	err := row.Scan(&todo.ID, &todo.Body, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *TodoServiceImp) DeleteByID(id int) error {
	stmt := "DELETE FROM todos WHERE id = $1"
	_, err := s.db.Exec(stmt, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *TodoServiceImp) Update(id int, body string) (*Todo, error) {
	stmt := "UPDATE todos SET todo = $2 WHERE id = $1"
	_, err := s.db.Exec(stmt, id, body)
	if err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

type Server struct {
	db      *sql.DB
	service TodoService
}

type Todo struct {
	ID        int64     `json:"id"`
	Body      string    `json:"todo"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Server) All(c *gin.Context) {
	todos, err := s.service.All()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("db: query error: %s", err),
		})
		return
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

	if err := s.service.Insert(&todo); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, todo)
}

func (s *Server) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	todo, err := s.service.GetByID(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, todo)
}

func (s *Server) DeleteByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := s.service.DeleteByID(id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) Update(c *gin.Context) {
	h := map[string]string{}
	if err := c.ShouldBindJSON(&h); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	todo, err := s.service.Update(id, h["todo"])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, todo)
}

func setupRoute(s *Server) *gin.Engine {
	r := gin.Default()
	// r.Use(gin.BasicAuth(gin.Accounts{
	// 	"foo": "bar",
	// }))
	r.Use(func(c *gin.Context) {
		if user, pass, ok := c.Request.BasicAuth(); ok {
			if user == "foo" && pass == "bar" {
				c.Set(gin.AuthUserKey, user)
				return
			}
		}

		c.AbortWithStatus(http.StatusUnauthorized)
	})
	r.GET("/todos", s.All)
	r.POST("/todos", s.Create)

	r.GET("/todos/:id", s.GetByID)
	r.PUT("/todos/:id", s.Update)
	r.DELETE("/todos/:id", s.DeleteByID)
	return r
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
	CREATE TABLE IF NOT EXISTS secrets (
		id SERIAL PRIMARY KEY,
		key TEXT
	);
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}

	s := &Server{
		service: &TodoServiceImp{
			db: db,
		},
	}

	r := setupRoute(s)

	r.Run(":" + os.Getenv("PORT"))
}
