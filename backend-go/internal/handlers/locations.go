package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LocationHandler struct {
	db *sql.DB
}

func NewLocationHandler(db *sql.DB) *LocationHandler {
	return &LocationHandler{db: db}
}

func (h *LocationHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Name        string  `json:"name"     binding:"required"`
		LocalName   string  `json:"local_name"`
		Description string  `json:"description"`
		Category    string  `json:"category"`
		Lat         float64 `json:"lat"      binding:"required"`
		Lng         float64 `json:"lng"      binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Category == "" {
		req.Category = "place"
	}

	id          := uuid.New().String()
	addressCode := generateAddressCode(req.Lat, req.Lng)
	now         := time.Now()

	_, err := h.db.Exec(`
		INSERT INTO locations
			(id, name, local_name, description, category, geom, address_code, created_by, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($7, $6), 4326), $8, $9, $10, $10)
	`, id, req.Name, req.LocalName, req.Description, req.Category,
		req.Lat, req.Lng, addressCode, userID, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create location"})
		return
	}

	h.db.Exec(`UPDATE users SET reputation = reputation + 10 WHERE id = $1`, userID)

	c.JSON(http.StatusCreated, gin.H{
		"id":           id,
		"address_code": addressCode,
		"message":      "Location created successfully",
	})
}

func (h *LocationHandler) List(c *gin.Context) {
	lat  := parseQueryFloat(c, "lat",    -9.4313)
	lng  := parseQueryFloat(c, "lng",    160.0521)
	radius := parseQueryFloat(c, "radius", 5000)

	rows, err := h.db.Query(`
		SELECT id, name, local_name, description, category,
			ST_Y(geom) AS lat, ST_X(geom) AS lng,
			address_code, created_by, upvotes, verified, created_at
		FROM locations
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			$3
		)
		ORDER BY upvotes DESC, created_at DESC
		LIMIT 200
	`, lat, lng, radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()

	type Location struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		LocalName   string    `json:"local_name,omitempty"`
		Description string    `json:"description,omitempty"`
		Category    string    `json:"category"`
		Lat         float64   `json:"lat"`
		Lng         float64   `json:"lng"`
		AddressCode string    `json:"address_code"`
		CreatedBy   string    `json:"created_by"`
		Upvotes     int       `json:"upvotes"`
		Verified    bool      `json:"verified"`
		CreatedAt   time.Time `json:"created_at"`
	}

	locations := []Location{}
	for rows.Next() {
		var loc Location
		var localName, description, addressCode sql.NullString
		err := rows.Scan(
			&loc.ID, &loc.Name, &localName, &description, &loc.Category,
			&loc.Lat, &loc.Lng, &addressCode, &loc.CreatedBy,
			&loc.Upvotes, &loc.Verified, &loc.CreatedAt,
		)
		if err != nil {
			continue
		}
		loc.LocalName   = localName.String
		loc.Description = description.String
		loc.AddressCode = addressCode.String
		locations = append(locations, loc)
	}
	c.JSON(http.StatusOK, gin.H{"locations": locations, "count": len(locations)})
}

func (h *LocationHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var name, category, createdBy string
	var localName, description, addressCode sql.NullString
	var lat, lng float64
	var upvotes int
	var verified bool
	var createdAt time.Time

	err := h.db.QueryRow(`
		SELECT id, name, local_name, description, category,
			ST_Y(geom), ST_X(geom), address_code,
			created_by, upvotes, verified, created_at
		FROM locations WHERE id = $1
	`, id).Scan(
		&id, &name, &localName, &description, &category,
		&lat, &lng, &addressCode,
		&createdBy, &upvotes, &verified, &createdAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "name": name,
		"local_name": localName.String, "description": description.String,
		"category": category, "lat": lat, "lng": lng,
		"address_code": addressCode.String,
		"created_by": createdBy, "upvotes": upvotes,
		"verified": verified, "created_at": createdAt,
	})
}

func (h *LocationHandler) Update(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	var req struct {
		Name        string `json:"name"        binding:"required"`
		LocalName   string `json:"local_name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.db.Exec(`
		UPDATE locations SET name=$1, local_name=$2, description=$3, updated_at=NOW()
		WHERE id=$4 AND created_by=$5
	`, req.Name, req.LocalName, req.Description, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not found or unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Location updated"})
}

func (h *LocationHandler) Delete(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	result, err := h.db.Exec(
		`DELETE FROM locations WHERE id=$1 AND created_by=$2`, id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not found or unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Location deleted"})
}

func (h *LocationHandler) Upvote(c *gin.Context) {
	id     := c.Param("id")
	userID := c.GetString("user_id")
	h.db.Exec(`
		INSERT INTO location_upvotes (user_id, location_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, id)
	h.db.Exec(`UPDATE locations SET upvotes = upvotes + 1 WHERE id = $1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "Upvoted"})
}

func generateAddressCode(lat, lng float64) string {
	area  := latLngToAreaCode(lat, lng)
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code  := make([]byte, 4)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("SLB-%s-%s", area, string(code))
}

func latLngToAreaCode(lat, lng float64) string {
	areas := map[string][4]float64{
		"HON": {-9.55, -9.35, 159.9, 160.1},
		"MAL": {-9.1, -8.2, 160.5, 161.5},
		"WES": {-8.5, -7.0, 156.5, 157.5},
		"MAK": {-10.6, -10.2, 161.8, 162.1},
	}
	for code, bb := range areas {
		if lat >= bb[0] && lat <= bb[1] && lng >= bb[2] && lng <= bb[3] {
			return code
		}
	}
	return strings.ToUpper(fmt.Sprintf("%c%c%c",
		'A'+int(-lat)%26,
		'A'+int(lng)%26,
		'A'+int((lat+lng)*-10)%26,
	))
}

func parseQueryFloat(c *gin.Context, key string, defaultVal float64) float64 {
	var f float64
	if _, err := fmt.Sscanf(c.DefaultQuery(key, fmt.Sprintf("%f", defaultVal)), "%f", &f); err != nil {
		return defaultVal
	}
	return f
}
