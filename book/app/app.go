package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetUp(cfg *Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.DBName)
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{})

	if err != nil {
		return err
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	app := &App{
		cfg:    cfg,
		db:     db,
		engine: engine,
	}

	return app.Start()
}

type Book struct {
	gorm.Model

	Name    string `json:"name"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

type App struct {
	cfg *Config
	db  *gorm.DB

	engine *gin.Engine
}

func (app *App) installAPI() error {
	group := app.engine.Group("/v1/books")

	group.POST("/", app.createBook)
	group.GET("/", app.listBooks)
	group.GET("/:id", app.getBook)
	group.PATCH("/:id", app.updateBook)
	group.DELETE("/:id", app.deleteBook)

	return nil
}

type ListBookReq struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`
}

func (app *App) listBooks(ctx *gin.Context) {
	var req ListBookReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	books := make([]Book, 0)
	page := req.Page
	if page == 0 {
		page = 1
	}
	size := req.Size
	if size == 0 {
		size = 10
	}
	err := app.db.Model(&Book{}).Offset((page - 1) * size).Limit(size).Find(&books).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": books})
}

func (app *App) getBook(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)

	book := &Book{}
	err := app.db.Where("id = ?", id).First(book).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": book})
}

type CreateBookReq struct {
	Name    string `form:"name" json:"name"`
	Author  string `form:"author" json:"author"`
	Content string `form:"content" json:"content"`
}

func (app *App) createBook(ctx *gin.Context) {
	var req CreateBookReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	book := &Book{
		Name:    req.Name,
		Author:  req.Author,
		Content: req.Content,
	}

	err := app.db.Create(book).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": book})
}

type UpdateBookReq struct {
	Name    string `form:"name" json:"name"`
	Author  string `form:"author" json:"author"`
	Content string `form:"content" json:"content"`
}

func (app *App) updateBook(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)

	var req UpdateBookReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := app.db.Where("id = ?", id).Updates(req).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	book := &Book{}
	err = app.db.Where("id = ?", id).First(book).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": req})
}

func (app *App) deleteBook(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)

	err := app.db.Where("id = ?", id).Delete(&Book{}).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": "OK"})
}

func (app *App) Start() error {
	if err := app.db.AutoMigrate(Book{}); err != nil {
		return err
	}

	if err := app.installAPI(); err != nil {
		return err
	}

	serve := http.Server{
		Addr:              app.cfg.Address,
		Handler:           app.engine,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		if err := serve.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	select {
	case <-ch:
	}

	if err := serve.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
