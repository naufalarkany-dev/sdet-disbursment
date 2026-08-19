package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"example.com/disbursement/internal/handlers"
	"example.com/disbursement/internal/middleware"
	"example.com/disbursement/internal/repository/memory"
	"example.com/disbursement/internal/services"
)

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-secret-key"
	}

	repo := memory.NewDisbursementRepository()
	svc := services.NewDisbursementService(repo)
	h := handlers.NewHandler(svc)

	r := gin.Default()
	SetupRoutes(r, h, jwtSecret)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// SetupRoutes mendaftarkan semua route ke router.
// Fungsi ini diekspos agar bisa digunakan di integration test.
func SetupRoutes(r *gin.Engine, h *handlers.Handler, jwtSecret string) {
	auth := r.Group("/")
	auth.Use(middleware.JWTAuth(jwtSecret))
	{
		auth.POST("/disbursements", h.CreateDisbursement)
		auth.PATCH("/disbursements/:id/status", h.UpdateDisbursementStatus)
		auth.GET("/disbursements", h.ListDisbursements)
	}
}
