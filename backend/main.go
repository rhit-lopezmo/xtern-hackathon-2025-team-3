package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	router := gin.Default()
	router.POST("/chat", handleChat)
	if err := router.Run("localhost:8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
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
				"role":    "system",
				"content": "You are a friendly, concise programming assistant who explains concepts with examples when helpful.",
			},
			{
				"role":    "user",
				"content": input.Prompt,
			},
		},
		"max_tokens":  100,
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
	if err := json.Unmarshal(body, &aiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse OpenAI response"})
		return
	}
	if len(aiResp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no response from OpenAI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": aiResp.Choices[0].Message.Content,
	})
}
