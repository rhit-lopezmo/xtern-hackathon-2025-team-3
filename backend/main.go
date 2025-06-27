package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type promptRequest struct {
	Prompt string `json:"prompt"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type recipe struct {
	Title       string   `json:"title"`
	InspiredBy  string   `json:"inspired_by"`
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	router := gin.Default()
	router.Static("/", "../frontend")
	router.POST("/chat", handleChat)

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server:", err)
		}
	}()
	log.Println("Server started on http://localhost:8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited cleanly")
}

func handleChat(c *gin.Context) {
	var input promptRequest
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OpenAI API key not set"})
		return
	}

	payload := map[string]interface{}{
		"model": "gpt-4-1106-preview",
		"messages": []map[string]string{
			{
				"role": "system",
				"content": `You are a friendly, confident home cooking assistant helping elderly users in Indiana recreate meals inspired by local restaurants and international cuisines.

Users will describe the kind of food they want. Your job is to generate a list of 3–5 recipes that match, inspired by popular Indiana restaurants or international cuisines (e.g. Hacienda, Pizza King, Thai, Indian).

ALWAYS return a JSON array of recipe objects using this format:
[
  {
    "title": "Dish name",
    "inspired_by": "Restaurant or cuisine + dish name",
    "ingredients": ["List of ingredients"],
    "steps": ["Step-by-step instructions"]
  },
  ...
]

NEVER return just one object. NEVER leave any fields blank or null. If unsure, make a best guess.

Only return valid JSON. Do not wrap in markdown. Do not explain. Output just the array.`,
			},
			{
				"role":    "user",
				"content": input.Prompt,
			},
		},
		"max_tokens":  1000,
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode request"})
		return
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create OpenAI request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reach OpenAI"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read OpenAI response"})
		return
	}

	var aiResp openAIResponse
	if err := json.Unmarshal(body, &aiResp); err != nil || len(aiResp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse OpenAI wrapper"})
		return
	}

	// Parse the JSON string inside assistant's message.content as an array
	content := aiResp.Choices[0].Message.Content
	var parsed []recipe
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		log.Println("Failed to parse recipe list:", err)
		c.JSON(http.StatusOK, gin.H{
			"raw":   content,
			"error": "Response was not valid recipe list JSON",
		})
		return
	}

	c.JSON(http.StatusOK, parsed)
}
