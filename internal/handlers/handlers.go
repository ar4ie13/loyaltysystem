package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/handlers/config"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var errorStatusMap = map[error]int{
	apperrors.ErrBalanceNotEnough:       http.StatusPaymentRequired,
	apperrors.ErrUserAlreadyExists:      http.StatusConflict,
	apperrors.ErrNoOrders:               http.StatusNoContent,
	apperrors.ErrPasswordMinSymbols:     http.StatusBadRequest,
	apperrors.ErrOrderAlreadyExists:     http.StatusOK,
	apperrors.ErrIncorrectOrderNumber:   http.StatusUnprocessableEntity,
	apperrors.ErrOrderNumberAlreadyUsed: http.StatusConflict,
}

// Handlers is a main object for handlers layer
type Handlers struct {
	cfg  config.ServerConf
	auth Auth
	srv  Service
	zlog zerolog.Logger
}

// NewHandlers creates Handler object
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
	BuildJWTString(userUUID uuid.UUID) (string, error)
	ValidateUserUUID(tokenString string) (uuid.UUID, error)
	GenerateHashFromPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

// Service interface used in handlers layer
type Service interface {
	LoginUser(ctx context.Context, login string) (models.User, error)
	CreateUser(ctx context.Context, user models.User) error
	PutUserOrder(ctx context.Context, user uuid.UUID, order string) error
	GetUserOrders(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error)
	GetBalance(ctx context.Context, user uuid.UUID) (models.User, error)
	PutUserWithdrawnOrder(ctx context.Context, user uuid.UUID, orderNum string, withdrawn float64) error
	GetUserWithdrawals(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error)
}

// ListenAndServe starts server
func (h *Handlers) ListenAndServe() error {
	router := h.newRouter()

	h.zlog.Info().Msgf("listening on %v", h.cfg.ServerAddr)

	if err := router.Run(h.cfg.ServerAddr); err != nil {
		return err
	}

	return nil
}

// newRouter contains all routes used by server
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
		user.POST("/orders", h.postOrder)
		user.GET("/balance", h.getUserBalance)
		user.POST("/balance/withdraw", h.postOrderWithWithdrawn)
	}

	userGzip := router.Group("/api/user").Use(h.authMiddleware()).Use(h.gzipMiddleware())
	{
		userGzip.GET("/orders", h.getUserOrders)
		userGzip.GET("/withdrawals", h.getUserWithdrawals)
	}

	return router
}

// getStatusCode process error and return the correlated status code
func (h *Handlers) getStatusCode(err error) int {
	// fast error check
	if status, exists := errorStatusMap[err]; exists {
		return status
	}

	// For wrapped errors
	for errType, status := range errorStatusMap {
		if errors.Is(err, errType) {
			return status
		}
	}
	return http.StatusInternalServerError
}

// userRegister is a handler used for user registration by using provided login and password
func (h *Handlers) userRegister(c *gin.Context) {
	var registerReq registerRequest

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
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot generate hash from password",
			"details": err.Error(),
		})
		return
	}

	user := models.User{
		Login:        registerReq.Login,
		PasswordHash: passwordHash,
	}

	err = h.srv.CreateUser(c, user)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot create user",
			"details": err.Error(),
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
		"message": "user successfully registered",
		"login":   registerReq.Login,
	})
}

// userLogin is a handler used for users logging in
func (h *Handlers) userLogin(c *gin.Context) {
	var loginReq loginRequest

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
}

// testAuth used for testing authentication middleware
func (h *Handlers) testAuth(c *gin.Context) {
	userUUID, ok := c.Get("user_uuid")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "user orders: void",
		"user_uuid": userUUID,
	})
}

// getUserUUIDFromRequest is a helper that retrieves user UUID from request
func (h *Handlers) getUserUUIDFromRequest(c *gin.Context) (uuid.UUID, error) {
	user, ok := c.Get("user_uuid")
	if !ok {
		return uuid.Nil, errors.New("user uuid not found")
	}

	userUUID, err := uuid.Parse(user.(string))
	if err != nil {
		h.zlog.Debug().Msgf("cannot parse user UUID: %v", err)
		return uuid.Nil, err
	}

	return userUUID, nil
}

// postOrder is a handler used for posting order provided by user in request without withdrawn
func (h *Handlers) postOrder(c *gin.Context) {

	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	order, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cannot get order",
			"details": err.Error(),
		})
		return
	}

	err = h.srv.PutUserOrder(c, userUUID, string(order))
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot register order",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "order successfully registered",
		"order":   string(order),
	})
}

// getUserOrders is a handler that returns all user's orders
func (h *Handlers) getUserOrders(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	orders, err := h.srv.GetUserOrders(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get orders",
			"details": err.Error(),
		})
		return
	}

	var ordersResponse []userOrdersResponse
	for _, order := range orders {
		var orderResponse userOrdersResponse
		if order.Accrual != nil {
			accrual := float64(*order.Accrual) / 100.0
			orderResponse.Accrual = &accrual

		}
		orderResponse.OrderNumber = order.OrderNumber
		orderResponse.Status = order.Status
		orderResponse.CreatedAt = order.CreatedAt.Format(time.RFC3339)
		ordersResponse = append(ordersResponse, orderResponse)
	}
	c.JSON(http.StatusOK, ordersResponse)
}

// getUserBalance is a handler that return user's balance
func (h *Handlers) getUserBalance(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	balance, err := h.srv.GetBalance(c, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cannot get user balance",
			"details": err.Error(),
		})
		return
	}

	var userBal userBalance
	userBal.Balance = float64(balance.Balance) / 100
	userBal.Withdrawn = float64(balance.Withdrawn) / 100
	c.JSON(http.StatusOK, userBal)
}

// postOrderWithWithdrawn is a handler that post order with withdrawn
func (h *Handlers) postOrderWithWithdrawn(c *gin.Context) {

	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Bind JSON to struct
	var orderWWithdrawn orderWithWithdrawn
	if err = c.ShouldBindJSON(&orderWWithdrawn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	err = h.srv.PutUserWithdrawnOrder(c, userUUID, orderWWithdrawn.Order, orderWWithdrawn.Sum)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot register order",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "order successfully registered",
		"order":   orderWWithdrawn.Order,
	})
}

// getUserWithdrawals is a handler that returns all user's withdrawals
func (h *Handlers) getUserWithdrawals(c *gin.Context) {
	userUUID, err := h.getUserUUIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	orders, err := h.srv.GetUserWithdrawals(c, userUUID)
	if err != nil {
		c.JSON(h.getStatusCode(err), gin.H{
			"error":   "cannot get user balance",
			"details": err.Error(),
		})
		return
	}

	var ordersResponse []orderWithWithdrawn
	for _, order := range orders {
		var orderResponse orderWithWithdrawn
		orderResponse.Order = order.OrderNumber
		if order.Withdrawn != nil {
			orderResponse.Sum = float64(*order.Withdrawn) / 100
		}
		orderResponse.ProcessedAt = order.CreatedAt.Format(time.RFC3339)
		ordersResponse = append(ordersResponse, orderResponse)
	}
	c.JSON(http.StatusOK, ordersResponse)
}
