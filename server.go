package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// Create a new task
	r.POST("/api/tasks", func(c *gin.Context) {
		var task Task
		if err := c.ShouldBind(&task); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		task.ID = uuid.NewString()
		if task.ID == "" {
			c.JSON(500, gin.H{"error": "Failed to generate UUID"})
			return
		}
		task.Status = "to do"

		// Debugging output
		fmt.Printf("Task ID: %s\n", task.ID)
		fmt.Printf("Task Title: %s\n", task.Task)
		fmt.Printf("Task Description: %s\n", task.Description)
		fmt.Printf("Task DueDate: %s\n", task.DueDate)
		fmt.Printf("Task Status: %s\n", task.Status)

		res, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
			task.ID, task.Task, task.Description, task.DueDate, task.Status)
		if err != nil {
			log.Printf("Failed to insert task: %v\n", err)
			c.JSON(500, gin.H{"error": "Failed to create task"})
			return
		}

		rowsAffected, _ := res.RowsAffected()
		log.Printf("Successfully inserted task %s (Rows affected: %d)\n", task.ID, rowsAffected)

		c.HTML(http.StatusOK, "task.html", []Task{task})
	})

	// Get all tasks
	r.GET("/api/tasks", func(c *gin.Context) {
		tasks, err := getAllTasks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
			return
		}

		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "task.html", tasks)
			return
		}

		c.JSON(http.StatusOK, tasks)
	})

	// Get task by ID
	r.GET("/api/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")

		var task Task
		err := db.QueryRow("SELECT id, task, description, dueDate, status FROM tasks WHERE id = ?", id).
			Scan(&task.ID, &task.Task, &task.Description, &task.DueDate, &task.Status)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
			}
			return
		}

		c.JSON(http.StatusOK, task)
	})

	// Delete task by ID
	r.DELETE("/api/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
		if err != nil {
			log.Printf("Delete error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}

		log.Printf("Deleted task %s\n", id)
		c.Data(http.StatusOK, "text/html", []byte(""))
	})

	// Update task status by ID
	r.PUT("/api/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")

		var input struct {
			Status string `json:"status" form:"status"`
		}
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		if input.Status != "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status update"})
			return
		}

		_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", input.Status, id)
		if err != nil {
			log.Printf("Update error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}

		var updated Task
		err = db.QueryRow("SELECT id, task, description, dueDate, status FROM tasks WHERE id = ?", id).
			Scan(&updated.ID, &updated.Task, &updated.Description, &updated.DueDate, &updated.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated task"})
			return
		}

		c.HTML(http.StatusOK, "task.html", []Task{updated})
	})

	fmt.Println("Server running on http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}
