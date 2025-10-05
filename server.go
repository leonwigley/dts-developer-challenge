package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

	"github.com/joho/godotenv"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
    ID          string    `form:"id" json:"id"`
    Task        string    `form:"task" json:"task"`
    Description string    `form:"description" json:"description"`
    DueDate     time.Time `form:"dueDate" json:"dueDate"` 
    Status      string    `form:"status" json:"status"`
}

var db *pgxpool.Pool

func initDB() {
    var err error
    connString := os.Getenv("PG_URL")
    db, err = pgxpool.New(context.Background(), connString)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
        os.Exit(1)
    }

    err = db.Ping(context.Background())
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
        db.Close()
        os.Exit(1)
    }

    createTableSQL := `
    CREATE TABLE IF NOT EXISTS tasks (
        id TEXT PRIMARY KEY,
        task TEXT NOT NULL,
        description TEXT,
        dueDate TIMESTAMP,
        status TEXT NOT NULL
    );
    `
    _, err = db.Exec(context.Background(), createTableSQL)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create table: %v\n", err)
        db.Close()
        os.Exit(1)
    }

    fmt.Println("Database initialized successfully")
}

func getAllTasks() ([]Task, error) {
    if db == nil {
        return nil, fmt.Errorf("database connection is nil")
    }

    rows, err := db.Query(context.Background(), "SELECT id, task, description, dueDate, status FROM tasks")
    if err != nil {
        return nil, fmt.Errorf("query failed: %v", err)
    }
    defer rows.Close()

    var tasks []Task
    for rows.Next() {
        var t Task
        if err := rows.Scan(&t.ID, &t.Task, &t.Description, &t.DueDate, &t.Status); err != nil {
            return nil, fmt.Errorf("scan failed: %v", err)
        }
        tasks = append(tasks, t)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration failed: %v", err)
    }
    return tasks, nil
}

func main() {
	godotenv.Load()
    initDB()
    defer db.Close()

    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    r.LoadHTMLGlob("public/*.html")
    r.Static("/assets", "./public")

    r.GET("/", func(c *gin.Context) {
        tasks, err := getAllTasks()
        if err != nil {
            log.Printf("Failed to fetch tasks: %v", err)
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

        task.ID = uuid.NewString()
        if task.ID == "" {
            c.JSON(500, gin.H{"error": "Failed to generate UUID"})
            return
        }
        task.Status = "to do"

        fmt.Printf("Task ID: %s\n", task.ID)
        fmt.Printf("Task Title: %s\n", task.Task)
        fmt.Printf("Task Description: %s\n", task.Description)
        fmt.Printf("Task DueDate: %v\n", task.DueDate)
        fmt.Printf("Task Status: %s\n", task.Status)

        res, err := db.Exec(context.Background(), "INSERT INTO tasks (id, task, description, dueDate, status) VALUES ($1, $2, $3, $4, $5)",
            task.ID, task.Task, task.Description, task.DueDate, task.Status)
        if err != nil {
            log.Printf("Failed to insert task: %v\n", err)
            c.JSON(500, gin.H{"error": "Failed to create task"})
            return
        }

        rowsAffected := res.RowsAffected()
        log.Printf("Successfully inserted task %s (Rows affected: %d)\n", task.ID, rowsAffected)

        c.HTML(http.StatusOK, "task.html", []Task{task})
    })

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

    r.GET("/api/tasks/:id", func(c *gin.Context) {
        id := c.Param("id")

        var task Task
        err := db.QueryRow(context.Background(), "SELECT id, task, description, dueDate, status FROM tasks WHERE id = $1", id).
            Scan(&task.ID, &task.Task, &task.Description, &task.DueDate, &task.Status)

        if err != nil {
            if err == pgx.ErrNoRows {
                c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
            }
            return
        }

        c.JSON(http.StatusOK, task)
    })

    r.DELETE("/api/tasks/:id", func(c *gin.Context) {
        id := c.Param("id")

        result, err := db.Exec(context.Background(), "DELETE FROM tasks WHERE id = $1", id)
        if err != nil {
            log.Printf("Delete error: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
            return
        }

        rowsAffected := result.RowsAffected()
        if rowsAffected == 0 {
            c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
            return
        }

        log.Printf("Deleted task %s\n", id)
        c.Data(http.StatusOK, "text/html", []byte(""))
    })

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

        _, err := db.Exec(context.Background(), "UPDATE tasks SET status = $1 WHERE id = $2", input.Status, id)
        if err != nil {
            log.Printf("Update error: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
            return
        }

        var updated Task
        err = db.QueryRow(context.Background(), "SELECT id, task, description, dueDate, status FROM tasks WHERE id = $1", id).
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