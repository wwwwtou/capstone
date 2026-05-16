package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Standard Response Structure as per Phase 1 Requirements
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id"`
}

func main() {
	r := gin.Default()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// API v1 Group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/recommendations", func(c *gin.Context) {
			userID := c.DefaultQuery("user_id", "guest")
			
			// Mocking successful response for demo
			c.JSON(http.StatusOK, APIResponse{
				Code:    200,
				Message: "success",
				TraceID: "uuid-trace-12345",
				Data: gin.H{
					"videos": []gin.H{
						{"video_id": "v_01", "score": 0.98, "reason": "interest_match_tech"},
						{"video_id": "v_03", "score": 0.75, "reason": "globally_popular"},
					},
					"user_id": userID,
				},
			})
		})

		// Config endpoint (Requires JWT in production)
		v1.GET("/configs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"strategy_name": "engagement",
				"weight":        0.85,
				"is_active":      true,
			})
		})
	}

	log.Printf("TikTok Glocal Backend serving on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
