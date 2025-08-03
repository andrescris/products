package models

import "time"

// Product define la estructura de datos para un producto en la tienda.
type Product struct {
	ID          string                 `json:"id" firestore:"id"`
	SKU         string                 `json:"sku" firestore:"sku"`
	Name        string                 `json:"name" firestore:"name"`
	Description string                 `json:"description" firestore:"description"`
	Brand       string                 `json:"brand" firestore:"brand"`
	Category    string                 `json:"category" firestore:"category"`
	Barcode     string                 `json:"barcode" firestore:"barcode"`
	Weight      float64                `json:"weight" firestore:"weight"`
	Dimensions  map[string]float64     `json:"dimensions" firestore:"dimensions"`
	Price       float64                `json:"price" firestore:"price"`
	Currency    string                 `json:"currency" firestore:"currency"`
	ImageURL    string                 `json:"imageUrl" firestore:"imageUrl"`
	Metadata    map[string]interface{} `json:"metadata" firestore:"metadata"`
	CreatedAt   time.Time              `json:"createdAt" firestore:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" firestore:"updatedAt"`
	Active      bool                   `json:"active" firestore:"active"`
	// --- Campos Añadidos ---
	ProjectID   string                 `json:"project_id" firestore:"project_id"`
	Subdomain   string                 `json:"subdomain" firestore:"subdomain"`
}