// En pkg/models/product.go

package models

import "time"

// Variation no cambia.
type Variation struct {
	ID         string            `json:"id" firestore:"id"`
	SKU        string            `json:"sku" firestore:"sku"`
	Barcode    string            `json:"barcode,omitempty" firestore:"barcode,omitempty"`
	Price      float64           `json:"price" firestore:"price"`
	ImageURL   string            `json:"imageUrl,omitempty" firestore:"imageUrl,omitempty"`
	Stock      int               `json:"stock" firestore:"stock"` 
	Attributes map[string]string `json:"attributes" firestore:"attributes"`
	Active     bool              `json:"active" firestore:"active"`
}

// Product ahora puede ser simple O tener variaciones.
type Product struct {
	ID          string    `json:"id" firestore:"id"`
	Name        string    `json:"name" firestore:"name"`
	Description string    `json:"description" firestore:"description"`
	Brand       string    `json:"brand,omitempty" firestore:"brand,omitempty"`
	Category    string    `json:"category" firestore:"category"`
	Currency    string    `json:"currency" firestore:"currency"`
	Active      bool      `json:"active" firestore:"active"`
	ProjectID   string    `json:"project_id" firestore:"project_id"`
	Subdomain   string    `json:"subdomain" firestore:"subdomain"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" firestore:"updatedAt"`
	FilterPrice float64 `json:"filter_price" firestore:"filter_price"`

	// --- CAMPOS PARA PRODUCTO SIMPLE ---
	// Estos campos se usan si el array 'variations' está vacío.
	SKU      string  `json:"sku,omitempty" firestore:"sku,omitempty"`
	Price    float64 `json:"price,omitempty" firestore:"price,omitempty"`
	Stock    int     `json:"stock,omitempty" firestore:"stock,omitempty"`
	Barcode  string  `json:"barcode,omitempty" firestore:"barcode,omitempty"`
	ImageURL string  `json:"imageUrl,omitempty" firestore:"imageUrl,omitempty"`

	// --- CAMPO PARA VARIACIONES ---
	Variations []Variation `json:"variations,omitempty" firestore:"variations,omitempty"`

	// Otros campos que ya tenías
	Weight     float64                `json:"weight,omitempty" firestore:"weight,omitempty"`
	Dimensions map[string]float64     `json:"dimensions,omitempty" firestore:"dimensions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" firestore:"metadata,omitempty"`
}