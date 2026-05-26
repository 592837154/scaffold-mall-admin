package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Goods struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null;index"`
	Category  string    `json:"category" gorm:"size:30;not null"`
	Price     float64   `json:"price" gorm:"type:decimal(10,2);not null"`
	Status    string    `json:"status" gorm:"size:20;not null;index"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GoodsInput struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name" binding:"required"`
	Category string  `json:"category" binding:"required"`
	Price    float64 `json:"price" binding:"required,gte=0"`
	Status   string  `json:"status" binding:"required"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ListData struct {
	List  []Goods `json:"list"`
	Total int64   `json:"total"`
}

var db *gorm.DB

func main() {
	initDB()

	router := gin.Default()
	router.Use(corsMiddleware())

	api := router.Group("/api/goods")
	{
		api.GET("/list", listGoods)
		api.POST("/create", createGoods)
		api.PUT("/update", updateGoods)
		api.DELETE("/delete", deleteGoods)
	}

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}

func initDB() {
	var err error

	dsn := mysqlDSN()
	for i := 1; i <= 30; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		fmt.Printf("waiting for mysql, retry %d/30: %v\n", i, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(&Goods{}); err != nil {
		panic(err)
	}

	seedGoods()
}

func mysqlDSN() string {
	host := env("MYSQL_HOST", "127.0.0.1")
	port := env("MYSQL_PORT", "3306")
	user := env("MYSQL_USER", "mall")
	password := env("MYSQL_PASSWORD", "mall123456")
	database := env("MYSQL_DATABASE", "mall_admin")
	tls := env("MYSQL_TLS", "false")

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
		user,
		password,
		host,
		port,
		database,
		tls,
	)
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func seedGoods() {
	var count int64
	db.Model(&Goods{}).Count(&count)
	if count > 0 {
		return
	}

	categories := []string{"digital", "clothing", "food"}
	names := []string{"Wireless Headphones", "Shell Jacket", "Nut Gift Box", "Mechanical Keyboard", "Cotton Hoodie", "Low Fat Cookies"}

	goods := make([]Goods, 0, 46)
	for i := 0; i < 46; i++ {
		status := "on"
		if i%4 == 0 {
			status = "off"
		}

		goods = append(goods, Goods{
			Name:      names[i%len(names)] + " " + strconv.Itoa(i+1),
			Category:  categories[i%len(categories)],
			Price:     float64(6900+i*1280) / 100,
			Status:    status,
			CreatedAt: time.Now().AddDate(0, 0, -i),
			UpdatedAt: time.Now().AddDate(0, 0, -i),
		})
	}

	if err := db.Create(&goods).Error; err != nil {
		panic(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func listGoods(c *gin.Context) {
	current := parsePositiveInt(c.Query("current"), 1)
	pageSize := parsePositiveInt(c.Query("pageSize"), 10)
	name := strings.TrimSpace(c.Query("name"))
	status := strings.TrimSpace(c.Query("status"))

	query := db.Model(&Goods{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var list []Goods
	offset := (current - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, ListData{
		List:  list,
		Total: total,
	})
}

func createGoods(c *gin.Context) {
	var input GoodsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	goods := Goods{
		Name:     strings.TrimSpace(input.Name),
		Category: input.Category,
		Price:    input.Price,
		Status:   input.Status,
	}

	if err := db.Create(&goods).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, goods)
}

func updateGoods(c *gin.Context) {
	var input GoodsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if input.ID == 0 {
		fail(c, http.StatusBadRequest, "id is required")
		return
	}

	var goods Goods
	if err := db.First(&goods, input.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "goods not found")
			return
		}
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	goods.Name = strings.TrimSpace(input.Name)
	goods.Category = input.Category
	goods.Price = input.Price
	goods.Status = input.Status

	if err := db.Save(&goods).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, goods)
}

func deleteGoods(c *gin.Context) {
	id := parseID(c)
	if id == 0 {
		fail(c, http.StatusBadRequest, "id is required")
		return
	}

	result := db.Delete(&Goods{}, id)
	if result.Error != nil {
		fail(c, http.StatusInternalServerError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "goods not found")
		return
	}

	ok(c, gin.H{"id": id})
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseID(c *gin.Context) uint {
	if queryID := c.Query("id"); queryID != "" {
		id, _ := strconv.ParseUint(queryID, 10, 64)
		return uint(id)
	}

	var body struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return 0
	}

	return body.ID
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, Response{
		Success: false,
		Error:   msg,
	})
}
