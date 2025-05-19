package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	ID          string `form:"id" json:"id"`
	Task        string `form:"task" json:"task"`
	Description string `form:"description" json:"description"`
	DueDate     string `form:"dueDate" json:"dueDate"`
	Status      string `form:"status" json:"status"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data/tasks.db")
	if err != nil {
		log.Fatalf("SQLite connection failed: %v", err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("Failed to enable WAL: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			task TEXT NOT NULL,
			description TEXT,
			dueDate TEXT,
			status TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Connected to SQLite with WAL enabled.")
}

func getAllTasks() ([]Task, error) {
	rows, err := db.Query("SELECT id, task, description, dueDate, status FROM tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Task, &t.Description, &t.DueDate, &t.Status); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func main() {
	initDB()
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.LoadHTMLGlob("public/*.html")
	r.Static("/assets", "./public")

	r.GET("/", func(c *gin.Context) {
		tasks, err := getAllTasks()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch tasks")
			return
		}
		c.HTML(http.StatusOK, "index.html", gin.H{"Tasks": tasks})
	})

	r.POST("/api/tasks", func(c *gin.Context) {
		var task Task
		if err := c.ShouldBind(&task); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		// Process task (e.g., save to database)
		fmt.Printf("Task ID: %s\n", task.ID)
		fmt.Printf("Task Title: %s\n", task.Task)
		fmt.Printf("Task Description: %s\n", task.Description)
		fmt.Printf("Task DueDate: %s\n", task.DueDate)
		fmt.Printf("Task Status: %s\n", task.Status)
		c.JSON(200, gin.H{"message": "Task created", "task": task})
	})

	// r.GET("/api/tasks", func(c *gin.Context) {
	// 	tasks, err := getAllTasks()
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, tasks)
	// })

	// r.PUT("/api/tasks/:id", func(c *gin.Context) {
	// 	id := c.Param("id")
	// 	var task Task
	// 	if err := c.ShouldBindJSON(&task); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
	// 		return
	// 	}

	// 	_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", task.Status, id)
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"status": "updated"})
	// })

	// r.DELETE("/api/tasks/:id", func(c *gin.Context) {
	// 	id := c.Param("id")
	// 	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	// })

	fmt.Println("Server running on http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}
