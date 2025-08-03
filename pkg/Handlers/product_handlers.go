package Handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/andrescris/firestore/lib/firebase/firestore"
	"github.com/andrescris/products/pkg/models"
	"github.com/gin-gonic/gin" // <-- CORRECCIÓN AQUÍ
	"github.com/google/uuid"
)

// --- Helper para Permisos ---
func isSubdomainAllowed(allowed []interface{}, target string) bool {
	for _, s := range allowed {
		if str, ok := s.(string); ok && str == target {
			return true
		}
	}
	return false
}

func getSubdomainsFromContext(c *gin.Context) ([]interface{}, bool) {
	data, exists := c.Get("allowed_subdomains")
	if !exists {
		log.Println("HANDLER ERROR: 'allowed_subdomains' not found in context!")
		return nil, false
	}
	subdomains, ok := data.([]interface{})
	if !ok {
		log.Printf("HANDLER ERROR: Type assertion for 'allowed_subdomains' failed. Actual type: %T", data)
		return nil, false
	}
	return subdomains, true
}


// --- Handlers ---

func CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	if product.Name == "" || product.Price <= 0 || product.ProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: name, price, and project_id are required."})
		return
	}

	allowedSubdomains, ok := getSubdomainsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not verify user permissions."})
		return
	}
	
	if !isSubdomainAllowed(allowedSubdomains, product.Subdomain) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You do not have permission to create resources in this subdomain."})
		return
	}

	product.ID = "prod-" + uuid.New().String()
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now
	product.Active = true

	var data map[string]interface{}
	jsonData, _ := json.Marshal(product)
	json.Unmarshal(jsonData, &data)

	ctx := context.Background()
	if err := firestore.CreateDocumentWithID(ctx, "products", product.ID, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Product created successfully", "data": product})
}

func UpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	ctx := context.Background()

	productDoc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	productSubdomain, _ := productDoc.Data["subdomain"].(string)

	allowedSubdomains, ok := getSubdomainsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not verify user permissions."})
		return
	}
	
	if !isSubdomainAllowed(allowedSubdomains, productSubdomain) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You do not have permission to modify resources in this subdomain."})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}
	delete(updates, "id")
	delete(updates, "createdAt")
	delete(updates, "subdomain") 

	updates["updatedAt"] = time.Now().UTC()
	if err := firestore.UpdateDocument(ctx, "products", productID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Product updated successfully"})
}

func DeleteProduct(c *gin.Context) {
	productID := c.Param("id")
	ctx := context.Background()

	productDoc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	
	productSubdomain, _ := productDoc.Data["subdomain"].(string)

	allowedSubdomains, ok := getSubdomainsFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not verify user permissions."})
		return
	}
	
	if !isSubdomainAllowed(allowedSubdomains, productSubdomain) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You do not have permission to modify resources in this subdomain."})
		return
	}

	updates := map[string]interface{}{
		"active":    false,
		"updatedAt": time.Now().UTC(),
	}
	
	if err := firestore.UpdateDocument(ctx, "products", productID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Product deactivated successfully"})
}