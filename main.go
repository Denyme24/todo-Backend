package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Todo struct {
	ID        int    `json:"id"`
	Completed bool   `json:"completed"`
	Body      string `json:"body"`
}

var db *pgx.Conn

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err = pgx.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME")))
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer db.Close(context.Background())

	app := fiber.New()
	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Adjust this to your Frontend origin
		AllowHeaders: "Content-Type",
	}))
	PORT := os.Getenv("PORT")

	app.Get("/", func(c *fiber.Ctx) error {
		rows, err := db.Query(context.Background(), "SELECT id, completed, body FROM todos")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to query todos"})
		}
		defer rows.Close()

		var todos []Todo
		for rows.Next() {
			var todo Todo
			if err := rows.Scan(&todo.ID, &todo.Completed, &todo.Body); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Failed to scan todo"})
			}
			todos = append(todos, todo)
		}
		return c.Status(200).JSON(todos)
	})

	app.Post("/todos", func(c *fiber.Ctx) error {
		todo := &Todo{}
		if err := c.BodyParser(todo); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
		}

		err := db.QueryRow(context.Background(), "INSERT INTO todos (completed, body) VALUES ($1, $2) RETURNING id", todo.Completed, todo.Body).Scan(&todo.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to insert todo"})
		}

		return c.Status(201).JSON(todo)
	})

	app.Patch("/todos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var todo Todo
		err := db.QueryRow(context.Background(), "UPDATE todos SET completed = NOT completed WHERE id = $1 RETURNING id, completed, body", id).Scan(&todo.ID, &todo.Completed, &todo.Body)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Todo not found"})
		}
		return c.Status(200).JSON(todo)
	})

	app.Delete("/todos/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		_, err := db.Exec(context.Background(), "DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Todo not found"})
		}
		return c.Status(200).JSON(fiber.Map{"success": "Todo deleted"})
	})

	log.Fatal(app.Listen(":" + PORT))
}
