package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/andrescris/firestore/lib/firebase/auth" // Asegúrate que esta ruta coincida con tu módulo de firestore
	"github.com/gin-gonic/gin"
)

// APIKeyAuthMiddleware verifica la API Key estática en las solicitudes.
func APIKeyAuthMiddleware() gin.HandlerFunc {
	requiredAPIKey := os.Getenv("API_KEY")
	if requiredAPIKey == "" {
		panic("CRITICAL: API_KEY environment variable is not defined.")
	}

	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-API-KEY")
		if subtle.ConstantTimeCompare([]byte(clientKey), []byte(requiredAPIKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing API Key."})
			return
		}
		c.Next()
	}
}

// SessionAuthMiddleware verifica una sesión de usuario válida a través de X-Session-ID.
// Es flexible: si la sesión existe, inyecta los datos del usuario; si no, permite continuar
// para que las rutas públicas puedan funcionar.
func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Session-ID")
		clientSubdomain := c.GetHeader("X-Client-Subdomain")

		if clientSubdomain != "" {
			c.Set("subdomain", clientSubdomain)
		}

		// Si no hay sesión, simplemente continuamos. Las rutas decidirán si requieren los datos.
		if sessionID == "" {
			c.Next()
			return
		}

		if clientSubdomain == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Client-Subdomain header is required when providing a session."})
			return
		}

		// 1. Validar la sesión usando tu librería de auth
		sessionInfo, err := auth.ValidateSession(context.Background(), sessionID)
		if err != nil || !sessionInfo.Active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session."})
			return
		}

		// 2. Si la sesión es válida, inyectamos los datos del usuario en el contexto
		// para que los handlers puedan usarlos.
		c.Set("uid", sessionInfo.UID)
		c.Set("claims", sessionInfo.Claims)
		c.Set("subdomain", clientSubdomain) // Usamos el subdomain que el cliente envía

		c.Next()
	}
}
