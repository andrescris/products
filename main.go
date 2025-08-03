package main

import (
	"log"
	"os"

	apiKeyMiddleware "github.com/andrescris/apiKeyService/pkg/middleware"
	"github.com/andrescris/firestore/lib/firebase"
	handlers "github.com/andrescris/products/pkg/Handlers"
	"github.com/andrescris/query-service/queryservice"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
}

func main() {
	// 1. Inicializa Firebase
	if err := firebase.InitFirebaseFromEnv(); err != nil {
		log.Fatalf("CRITICAL: Error initializing Firebase connection: %v", err)
	}
	defer firebase.Close()
	// 2. Obtiene el cliente de Firestore
	firestoreClient := firebase.GetFirestoreClient()
	if firestoreClient == nil {
		log.Fatalf("CRITICAL: Firestore client is nil immediately after initialization!")
	}
	
	r := gin.Default()

	// 3. Inyecta la dependencia globalmente
	r.Use(func(c *gin.Context) {
		c.Set("firestoreClient", firestoreClient)
		c.Next()
	})

	api := r.Group("/api/v1")
	{
		// Endpoint genérico para consultas, ahora también para productos
		api.POST("/collections/:collection/query",
			apiKeyMiddleware.AuthMiddleware("read:products"), // Protegido con permiso de lectura
			queryservice.ConditionalSubdomainFilterMiddleware(),
			queryservice.QueryHandler,
		)

		products := api.Group("/products")
		{
			// --- RUTAS DE ESCRITURA ---
			// Protegidas con el permiso "write:products"
			writeRoutes := products.Group("/")
			writeRoutes.Use(apiKeyMiddleware.AuthMiddleware("write:products"))
			{
				writeRoutes.POST("/", handlers.CreateProduct)
				writeRoutes.PATCH("/:id", handlers.UpdateProduct)
				writeRoutes.DELETE("/:id", handlers.DeleteProduct)
			}
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("🚀 Servidor de API de Productos iniciado en http://localhost:%s", port)
	r.Run(":" + port)
}