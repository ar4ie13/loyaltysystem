package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/handlers/config"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Handlers struct {
	cfg  config.ServerConf
	auth Auth
	srv  Service
	zlog zerolog.Logger
}

func NewHandlers(cfg config.ServerConf, auth Auth, srv Service, zlog zerolog.Logger) *Handlers {
	return &Handlers{
		cfg:  cfg,
		auth: auth,
		srv:  srv,
		zlog: zlog,
	}
}

// Auth used for authentication
type Auth interface {
	GenerateUserUUID() uuid.UUID
	BuildJWTString(userUUID uuid.UUID) (string, error)
	ValidateUserUUID(tokenString string) (uuid.UUID, error)
	GenerateHashFromPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

type Service interface {
	LoginUser(ctx context.Context, login string) (models.User, error)
	CreateUser(ctx context.Context, user models.User) error
}

func (h *Handlers) ListenAndServe() error {
	router := h.newRouter()

	h.zlog.Info().Msgf("listening on %v", h.cfg.ServerAddr)

	if err := router.Run(h.cfg.ServerAddr); err != nil {
		return err
	}

	return nil
}

func (h *Handlers) newRouter() *gin.Engine {
	router := gin.New()

	//middlewares for router
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	//API routes
	auth := router.Group("/api/user")
	{
		auth.POST("/register", h.userRegister)
		auth.POST("/login", h.userLogin)
	}
	user := router.Group("/api/user").Use(h.authMiddleware())
	{
		user.GET("/test", h.testAuth)
	}
	return router
}

func (h *Handlers) userRegister(c *gin.Context) {
	var registerReq models.RegisterRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Process the register data
	passwordHash, err := h.auth.GenerateHashFromPassword(registerReq.Password)
	if err != nil {
		if errors.Is(err, apperrors.ErrPasswordMinSymbols) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "cannot generate hash from password",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "cannot generate hash from password",
				"details": err.Error(),
			})
		}
		return
	}

	user := models.User{
		UUID:         h.auth.GenerateUserUUID(),
		Login:        registerReq.Login,
		PasswordHash: passwordHash,
	}

	err = h.srv.CreateUser(c, user)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "cannot create user",
				"details": err.Error(),
			})

		} else {

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "cannot create user",
				"details": err.Error(),
			})
		}
		return
	}

	tokenString, err := h.auth.BuildJWTString(user.UUID)
	if err != nil {
		h.zlog.Error().Msgf("error building JWT string: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("user_uuid", tokenString, 0, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "user successfully registered",
		"login":   registerReq.Login,
	})
	return
}

func (h *Handlers) userLogin(c *gin.Context) {
	var loginReq models.LoginRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Process the login data
	user, err := h.srv.LoginUser(c, loginReq.Login)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "cannot login",
			"details": err.Error(),
		})
		return
	}

	if !h.auth.CheckPasswordHash(loginReq.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": apperrors.ErrInvalidPassword.Error(),
		})
		return
	}

	tokenString, err := h.auth.BuildJWTString(user.UUID)
	if err != nil {
		h.zlog.Error().Msgf("error building JWT string: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("user_uuid", tokenString, 0, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "user successfully logged in",
		"login":   loginReq.Login,
	})

	return
}

func (h *Handlers) testAuth(c *gin.Context) {
	userUUID, ok := c.Get("user_uuid")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "user orders: void",
		"user_uuid": userUUID,
	})
	return
}
