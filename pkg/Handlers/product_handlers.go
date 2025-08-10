package Handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/andrescris/firestore/lib/firebase"
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

	// --- LÓGICA DE VALIDACIÓN HÍBRIDA ---
	if len(product.Variations) == 0 {
		// Es un producto simple: validar campos principales
		if product.Name == "" || product.SKU == "" || product.Price <= 0 || product.ProjectID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Para un producto simple, se requieren: name, sku, price y project_id."})
			return
		}
	} else {
		// Es un producto con variaciones: validar campos de cada variación
		if product.Name == "" || product.ProjectID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Para un producto con variaciones, se requieren: name y project_id."})
			return
		}
		for _, v := range product.Variations {
			if v.SKU == "" || v.Price <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cada variación debe tener un sku y un price válido.", "variation_sku": v.SKU})
				return
			}
		}
	}

	if len(product.Variations) == 0 {
		// Es un producto simple. El precio de filtro es el precio principal.
		product.FilterPrice = product.Price
	} else {
		// Es un producto con variaciones. Calculamos el precio mínimo.
		var minPrice float64 = -1
		for _, v := range product.Variations {
			if minPrice == -1 || v.Price < minPrice {
				minPrice = v.Price
			}
		}
		product.FilterPrice = minPrice
	}

	// VALIDACIÓN ACTUALIZADA:
	// Eliminamos la validación de 'price' porque ahora pertenece a las variaciones.
	// Mantenemos las validaciones para los campos que sí son del producto principal.
	if product.Name == "" || product.ProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: name and project_id are required."})
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

	// LÓGICA DE CREACIÓN:
	// Aquí, podrías incluso procesar una lista de variaciones si vinieran en la petición inicial.
	// Por ahora, nos aseguramos de que el slice de variaciones no sea nulo para evitar problemas.
	if product.Variations == nil {
		product.Variations = []models.Variation{}
	}
	// También podrías iterar sobre `product.Variations` aquí y asignarles un ID único si se envían en la creación.
	// Por ejemplo:
	for i := range product.Variations {
		product.Variations[i].ID = "var-" + uuid.New().String()
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

	// --- Esta parte de verificación de permisos sigue igual ---
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
	// --- Fin de la verificación de permisos ---

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}
	// Medida de seguridad: Eliminar campos que no deberían ser actualizables por esta vía.
	delete(updates, "id")
	delete(updates, "createdAt")
	delete(updates, "subdomain")
	delete(updates, "project_id")
	delete(updates, "variations") // ¡MUY IMPORTANTE! Evita que se borren las variaciones.

	updates["updatedAt"] = time.Now().UTC()
	if err := firestore.UpdateDocument(ctx, "products", productID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Product updated successfully"})
}

func GetProductByID(c *gin.Context) {
	// --- AÑADIMOS LA VERIFICACIÓN AL INICIO ---
	userSubdomain, userSubdomainExists := c.Get("subdomain")
	if !userSubdomainExists {
		// Si el que pregunta no tiene un subdominio, no puede ver ningún producto.
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Subdomain context is required."})
		return
	}
	// --- FIN DE LA VERIFICACIÓN INICIAL ---

	productID := c.Param("id")
	ctx := context.Background()

	doc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	productSubdomain, productSubdomainOk := doc.Data["subdomain"].(string)

	// Verificamos la pertenencia del producto al subdominio del usuario.
	if !productSubdomainOk || userSubdomain.(string) != productSubdomain {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this resource."})
		return
	}

	// El resto de la función no cambia...
	var product models.Product
	jsonData, err := json.Marshal(doc.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process product data (marshal)"})
		return
	}
	if err := json.Unmarshal(jsonData, &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse product data (unmarshal)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": product})
}

func ListProducts(c *gin.Context) {
	var options firebase.QueryOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body for filters", "details": err.Error()})
		return
	}

	subdomain, subdomainExists := c.Get("subdomain")

	// --- INICIO DE LA CORRECCIÓN FINAL DE SEGURIDAD ---

	if !subdomainExists {
		// Si NO hay un subdominio en el contexto, no se permite la consulta.
		// Devolvemos una respuesta exitosa pero con cero resultados.
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   0,
			"query":   options,
			"data":    []models.Product{}, // Array de productos vacío
		})
		return // ¡Muy importante! Detenemos la ejecución aquí.
	}

	// Si llegamos aquí, es porque sí existe un subdominio y lo forzamos.
	secureFilters := []firebase.QueryFilter{}
	for _, filter := range options.Filters {
		if filter.Field != "subdomain" {
			secureFilters = append(secureFilters, filter)
		}
	}
	secureFilters = append(secureFilters, firebase.QueryFilter{
		Field:    "subdomain",
		Operator: "==",
		Value:    subdomain.(string),
	})
	options.Filters = secureFilters

	// --- FIN DE LA CORRECCIÓN FINAL DE SEGURIDAD ---

	// El resto de la función no cambia...
	ctx := context.Background()
	docs, err := firestore.QueryDocuments(ctx, "products", options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query products", "details": err.Error()})
		return
	}

	var products []models.Product
	for _, doc := range docs {
		var product models.Product
		jsonData, _ := json.Marshal(doc.Data)
		json.Unmarshal(jsonData, &product)
		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(products),
		"query":   options,
		"data":    products,
	})
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

func CreateVariation(c *gin.Context) {
	productID := c.Param("id")
	ctx := context.Background()

	// 1. Obtener el producto principal
	doc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// --- INICIO DE LA CORRECCIÓN ---
	var product models.Product
	jsonData, err := json.Marshal(doc.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process product data (marshal)", "details": err.Error()})
		return
	}
	if err := json.Unmarshal(jsonData, &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse product data (unmarshal)", "details": err.Error()})
		return
	}
	// --- FIN DE LA CORRECCIÓN ---

	// 2. Obtener y validar la nueva variación del cuerpo de la petición
	var newVariation models.Variation
	if err := c.ShouldBindJSON(&newVariation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variation data", "details": err.Error()})
		return
	}

	if newVariation.SKU == "" || newVariation.Price <= 0 || len(newVariation.Attributes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required variation fields: sku, price, and attributes are required."})
		return
	}

	for _, v := range product.Variations {
		if v.SKU == newVariation.SKU {
			c.JSON(http.StatusConflict, gin.H{"error": "A variation with this SKU already exists for this product."})
			return
		}
	}

	// 3. Asignar un nuevo ID y añadir la variación al producto
	newVariation.ID = "var-" + uuid.New().String()
	newVariation.Active = true
	product.Variations = append(product.Variations, newVariation)
	product.UpdatedAt = time.Now().UTC()

	// 4. Guardar el producto actualizado en Firestore
	var data map[string]interface{}
	jsonUpdateData, _ := json.Marshal(product)
	json.Unmarshal(jsonUpdateData, &data)

	if err := firestore.UpdateDocument(ctx, "products", productID, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add variation", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Variation created successfully", "data": newVariation})
}

func UpdateVariation(c *gin.Context) {
	productID := c.Param("id")
	variationID := c.Param("variationId")
	ctx := context.Background()

	// 1. Obtener el producto principal
	doc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// --- INICIO DE LA CORRECCIÓN ---
	var product models.Product
	jsonData, err := json.Marshal(doc.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process product data (marshal)", "details": err.Error()})
		return
	}
	if err := json.Unmarshal(jsonData, &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse product data (unmarshal)", "details": err.Error()})
		return
	}
	// --- FIN DE LA CORRECCIÓN ---

	// 2. Obtener los datos a actualizar
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	// 3. Encontrar y actualizar la variación
	variationFound := false
	for i, v := range product.Variations {
		if v.ID == variationID {
			variationFound = true
			if price, ok := updates["price"].(float64); ok {
				product.Variations[i].Price = price
			}
			if stock, ok := updates["stock"].(float64); ok {
				product.Variations[i].Stock = int(stock)
			}
			if imageUrl, ok := updates["imageUrl"].(string); ok {
				product.Variations[i].ImageURL = imageUrl
			}
			break
		}
	}

	if !variationFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variation not found"})
		return
	}

	product.UpdatedAt = time.Now().UTC()

	// 4. Guardar el producto actualizado
	var data map[string]interface{}
	jsonUpdateData, _ := json.Marshal(product)
	json.Unmarshal(jsonUpdateData, &data)

	if err := firestore.UpdateDocument(ctx, "products", productID, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update variation", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Variation updated successfully"})
}

func DeleteVariation(c *gin.Context) {
	productID := c.Param("id")
	variationID := c.Param("variationId")
	ctx := context.Background()

	doc, err := firestore.GetDocument(ctx, "products", productID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// --- INICIO DE LA CORRECCIÓN ---
	var product models.Product
	jsonData, err := json.Marshal(doc.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process product data (marshal)", "details": err.Error()})
		return
	}
	if err := json.Unmarshal(jsonData, &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse product data (unmarshal)", "details": err.Error()})
		return
	}
	// --- FIN DE LA CORRECCIÓN ---

	variationFound := false
	for i, v := range product.Variations {
		if v.ID == variationID {
			product.Variations[i].Active = false
			variationFound = true
			break
		}
	}

	if !variationFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variation not found"})
		return
	}

	product.UpdatedAt = time.Now().UTC()

	var data map[string]interface{}
	jsonUpdateData, _ := json.Marshal(product)
	json.Unmarshal(jsonUpdateData, &data)

	if err := firestore.UpdateDocument(ctx, "products", productID, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate variation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Variation deactivated successfully"})
}
