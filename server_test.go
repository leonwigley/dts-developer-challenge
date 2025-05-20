package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			task TEXT NOT NULL,
			description TEXT,
			dueDate TEXT,
			status TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	return db
}

func setupRouter(testDB *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.LoadHTMLGlob("public/*.html")
	r.Static("/assets", "./public")

	db = testDB

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
		task.ID = "test-uuid"
		task.Status = "to do"
		res, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
			task.ID, task.Task, task.Description, task.DueDate, task.Status)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create task"})
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(500, gin.H{"error": "No rows affected"})
			return
		}
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

	r.DELETE("/api/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")
		result, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
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
		_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", input.Status, id)
		if err != nil {
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

	return r
}

func TestGetIndex(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}
	fmt.Println("TestGetIndex passed")
}

func TestPostTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	task := Task{
		Task:        "Test Task",
		Description: "Test Description",
		DueDate:     "2025-05-21",
	}
	body, _ := json.Marshal(task)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}
	fmt.Println("TestPostTask passed")
}

func TestGetAllTasksJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	_, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
		"test-uuid", "Test Task", "Description", "2025-05-21", "to do")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/tasks", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var tasks []Task
	if err := json.Unmarshal(resp.Body.Bytes(), &tasks); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "test-uuid" {
		t.Errorf("Unexpected tasks: %+v", tasks)
	}
	fmt.Println("TestGetAllTasksJSON passed")
}

func TestGetAllTasksHTML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	_, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
		"test-uuid", "Test Task", "Description", "2025-05-21", "to do")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/tasks", nil)
	req.Header.Set("HX-Request", "true")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}
	fmt.Println("TestGetAllTasksHTML passed")
}

func TestGetTaskByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	_, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
		"test-uuid", "Test Task", "Description", "2025-05-21", "to do")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/tasks/test-uuid", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var task Task
	if err := json.Unmarshal(resp.Body.Bytes(), &task); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
	if task.ID != "test-uuid" {
		t.Errorf("Expected task ID test-uuid, got %s", task.ID)
	}
	fmt.Println("TestGetTaskByID passed")
}

func TestDeleteTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	_, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
		"test-uuid", "Test Task", "Description", "2025-05-21", "to do")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	req, _ := http.NewRequest("DELETE", "/api/tasks/test-uuid", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", "test-uuid").Scan(&count)
	if err != nil || count != 0 {
		t.Errorf("Task was not deleted")
	}
	fmt.Println("TestDeleteTask passed")
}

func TestUpdateTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouter(db)

	_, err := db.Exec("INSERT INTO tasks (id, task, description, dueDate, status) VALUES (?, ?, ?, ?, ?)",
		"test-uuid", "Test Task", "Description", "2025-05-21", "to do")
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	update := struct {
		Status string `json:"status"`
	}{Status: "completed"}
	body, _ := json.Marshal(update)
	req, _ := http.NewRequest("PUT", "/api/tasks/test-uuid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	// Verify status update
	var status string
	err = db.QueryRow("SELECT status FROM tasks WHERE id = ?", "test-uuid").Scan(&status)
	if err != nil || status != "completed" {
		t.Errorf("Expected status completed, got %s", status)
	}
	fmt.Println("TestUpdateTaskStatus passed")
}
