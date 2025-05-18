package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
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
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Connected to SQLite with WAL enabled.")
}

func getAllTasks() ([]Task, error) {
	rows, err := db.Query("SELECT id, title, description, status FROM tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status); err != nil {
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

	r.GET("/", func(ctx *gin.Context) {
		tasks, err := getAllTasks()
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Failed to fetch tasks")
			return
		}
		ctx.HTML(http.StatusOK, "index.html", gin.H{"Tasks": tasks})
	})

	r.GET("/api/tasks", func(ctx *gin.Context) {
		tasks, err := getAllTasks()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
			return
		}
		ctx.JSON(http.StatusOK, tasks)
	})

	r.POST("/api/tasks", func(ctx *gin.Context) {
		var task Task
		if err := ctx.ShouldBindJSON(&task); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		task.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		task.Status = "pending"

		_, err := db.Exec("INSERT INTO tasks (id, title, description, status) VALUES (?, ?, ?, ?)",
			task.ID, task.Title, task.Description, task.Status,
		)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert task"})
			return
		}
		ctx.JSON(http.StatusCreated, task)
	})

	r.PUT("/api/tasks/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		var task Task
		if err := ctx.ShouldBindJSON(&task); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", task.Status, id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "updated"})
	})

	r.DELETE("/api/tasks/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})

	fmt.Println("Server running on http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}
