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

type DailySales struct {
	Date      string    `json:"date" gorm:"size:10;primaryKey"`
	Quantity  int       `json:"quantity" gorm:"not null;default:0"`
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

type DailySalesInput struct {
	Date      string `json:"date" binding:"required"`
	Quantity  int    `json:"quantity"`
	Increment int    `json:"increment"`
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

const (
	apiGoodsPath         = "/api/goods"
	apiSalesCalendarPath = "/api/sales-calendar"
	routeListPath        = "/list"
	routeCreatePath      = "/create"
	routeUpdatePath      = "/update"
	routeDeletePath      = "/delete"
	routeSavePath        = "/save"
	routeIncrementPath   = "/increment"
	queryCurrentKey      = "current"
	queryPageSizeKey     = "pageSize"
	queryNameKey         = "name"
	queryStatusKey       = "status"
	queryIDKey           = "id"
	queryYearKey         = "year"
	dateLayout           = "2006-01-02"
	serverAddress        = ":8080"
	sqlDateRangeFilter   = "date >= ? AND date <= ?"
	sqlDateEqualsFilter  = "date = ?"
	sqlDateAscOrder      = "date ASC"
	columnQuantity       = "quantity"
	errInvalidDate       = "invalid date"
	errQuantityRange     = "quantity must be between 0 and 5"
	errIncrementPositive = "increment must be positive"
	dailySalesMin        = 0
	dailySalesMax        = 5
	dbConnectRetries     = 30
	dbConnectRetryDelay  = 2 * time.Second
	dbMaxOpenConns       = 5
	dbMaxIdleConns       = 1
	dbConnMaxLifetime    = 2 * time.Minute
	dbConnMaxIdleTime    = 30 * time.Second
	mysqlDefaultHost     = "127.0.0.1"
	mysqlDefaultPort     = "3306"
	mysqlDefaultUser     = "mall"
	mysqlDefaultPassword = "mall123456"
	mysqlDefaultDatabase = "mall_admin"
	mysqlDefaultTLS      = "false"
	mysqlEnvHost         = "MYSQL_HOST"
	mysqlEnvPort         = "MYSQL_PORT"
	mysqlEnvUser         = "MYSQL_USER"
	mysqlEnvPassword     = "MYSQL_PASSWORD"
	mysqlEnvDatabase     = "MYSQL_DATABASE"
	mysqlEnvTLS          = "MYSQL_TLS"
	mysqlDSNFormat       = "%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s&timeout=5s&readTimeout=8s&writeTimeout=8s"
)

var db *gorm.DB

// main 初始化数据库、注册路由并启动 HTTP 服务。
// 没有入参和返回值。
// 副作用：连接数据库、执行表结构迁移、写入演示商品数据，并监听 8080 端口。
func main() {
	initDB()

	router := gin.Default()
	router.Use(corsMiddleware())

	api := router.Group(apiGoodsPath)
	{
		api.GET(routeListPath, listGoods)
		api.POST(routeCreatePath, createGoods)
		api.PUT(routeUpdatePath, updateGoods)
		api.DELETE(routeDeletePath, deleteGoods)
	}

	salesAPI := router.Group(apiSalesCalendarPath)
	{
		salesAPI.GET(routeListPath, listDailySales)
		salesAPI.PUT(routeSavePath, saveDailySales)
		salesAPI.POST(routeIncrementPath, incrementDailySales)
	}

	if err := router.Run(serverAddress); err != nil {
		panic(err)
	}
}

// initDB 建立数据库连接并执行数据表迁移。
// 没有入参和返回值。
// 副作用：最多重试连接数据库、配置连接池生命周期、迁移商品表和每日销量表，并在商品表为空时写入演示数据。
func initDB() {
	var err error

	dsn := mysqlDSN()
	for i := 1; i <= dbConnectRetries; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		fmt.Printf("waiting for mysql, retry %d/%d: %v\n", i, dbConnectRetries, err)
		time.Sleep(dbConnectRetryDelay)
	}

	if err != nil {
		panic(err)
	}

	configureDBPool()

	if err := db.AutoMigrate(&Goods{}, &DailySales{}); err != nil {
		panic(err)
	}

	seedGoods()
}

// configureDBPool 配置 MySQL 连接池。
// 没有入参和返回值。
// 副作用：限制连接池规模和空闲连接生命周期，避免 TiDB Cloud 断开空闲连接后后端继续复用坏连接。
func configureDBPool() {
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(dbMaxOpenConns)
	sqlDB.SetMaxIdleConns(dbMaxIdleConns)
	sqlDB.SetConnMaxLifetime(dbConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConnMaxIdleTime)
}

// mysqlDSN 生成 MySQL/TiDB 连接字符串。
// 没有入参。
// 返回值为 GORM MySQL 驱动使用的 DSN。
// 副作用：读取数据库相关环境变量；DSN 内设置连接、读、写超时，避免远程连接异常时接口长时间无响应。
func mysqlDSN() string {
	host := env(mysqlEnvHost, mysqlDefaultHost)
	port := env(mysqlEnvPort, mysqlDefaultPort)
	user := env(mysqlEnvUser, mysqlDefaultUser)
	password := env(mysqlEnvPassword, mysqlDefaultPassword)
	database := env(mysqlEnvDatabase, mysqlDefaultDatabase)
	tls := env(mysqlEnvTLS, mysqlDefaultTLS)

	return fmt.Sprintf(
		mysqlDSNFormat,
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

// listDailySales 查询指定年份内的每日销量。
// 参数通过 Gin 上下文读取：`year` 查询参数可选，缺省为当前年份。
// 返回值通过 JSON 写入响应，数据为每日销量列表。
// 副作用：只读取数据库，不修改状态。
func listDailySales(c *gin.Context) {
	year := parseYear(c.Query(queryYearKey), time.Now().Year())
	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local)

	var list []DailySales
	if err := db.
		Where(sqlDateRangeFilter, startDate.Format(dateLayout), endDate.Format(dateLayout)).
		Order(sqlDateAscOrder).
		Find(&list).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, list)
}

// saveDailySales 覆盖保存某一天的销量。
// 参数通过请求体读取：`date` 为日期，`quantity` 为 0 到 5 的当天销量。
// 返回值通过 JSON 写入响应，数据为保存后的每日销量记录。
// 副作用：向数据库新增或更新一条每日销量记录。
func saveDailySales(c *gin.Context) {
	var input DailySalesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	date, valid := normalizeSalesDate(input.Date)
	if !valid {
		fail(c, http.StatusBadRequest, errInvalidDate)
		return
	}

	if input.Quantity < dailySalesMin || input.Quantity > dailySalesMax {
		fail(c, http.StatusBadRequest, errQuantityRange)
		return
	}

	record := DailySales{Date: date}
	if err := db.Where(sqlDateEqualsFilter, date).
		Assign(map[string]interface{}{columnQuantity: input.Quantity}).
		FirstOrCreate(&record).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, record)
}

// incrementDailySales 按增量更新某一天的销量。
// 参数通过请求体读取：`date` 为日期，`increment` 为需要增加的数量。
// 返回值通过 JSON 写入响应，数据为更新后的每日销量记录。
// 副作用：在数据库中新增或更新一条每日销量记录，最终销量会限制在 0 到 5。
func incrementDailySales(c *gin.Context) {
	var input DailySalesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	date, valid := normalizeSalesDate(input.Date)
	if !valid {
		fail(c, http.StatusBadRequest, errInvalidDate)
		return
	}

	if input.Increment <= dailySalesMin {
		fail(c, http.StatusBadRequest, errIncrementPositive)
		return
	}

	var record DailySales
	if err := db.First(&record, sqlDateEqualsFilter, date).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		record = DailySales{Date: date}
	}

	record.Quantity += input.Increment
	if record.Quantity > dailySalesMax {
		record.Quantity = dailySalesMax
	}

	if err := db.Model(&record).Update(columnQuantity, record.Quantity).Error; err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, record)
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

// listGoods 查询商品分页列表。
// 参数通过 Gin 上下文读取：`current`、`pageSize` 控制分页，`name`、`status` 控制筛选。
// 返回值通过 JSON 写入响应，数据包含商品列表和总数。
// 副作用：只读取数据库，不修改商品数据。
func listGoods(c *gin.Context) {
	current := parsePositiveInt(c.Query(queryCurrentKey), 1)
	pageSize := parsePositiveInt(c.Query(queryPageSizeKey), 10)
	name := strings.TrimSpace(c.Query(queryNameKey))
	status := strings.TrimSpace(c.Query(queryStatusKey))

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

// parseYear 将查询字符串解析为年份。
// 参数 value 为年份字符串，fallback 为解析失败时使用的年份。
// 返回值为有效年份；无效输入会返回 fallback。
// 副作用：无。
func parseYear(value string, fallback int) int {
	year, err := strconv.Atoi(value)
	if err != nil || year <= 0 {
		return fallback
	}

	return year
}

// normalizeSalesDate 校验并规范化每日销量日期。
// 参数 value 为请求体中的日期字符串。
// 返回值为规范化后的 `YYYY-MM-DD` 日期和校验是否成功。
// 副作用：无。
func normalizeSalesDate(value string) (string, bool) {
	date, err := time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return "", false
	}

	return date.Format(dateLayout), true
}

// parseID 从请求查询参数或请求体中解析商品 id。
// 参数 c 为 Gin 请求上下文。
// 返回值为解析到的商品 id；解析失败时返回 0。
// 副作用：当查询参数没有 id 时会尝试读取 JSON 请求体，可能消耗请求体内容。
func parseID(c *gin.Context) uint {
	if queryID := c.Query(queryIDKey); queryID != "" {
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
