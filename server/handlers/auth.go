package handlers

import (
	"fmt"
	"main/auth"
	"main/helpers"
	"main/models"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

func (h *AuthHandler) Login(c *gin.Context) {
	type LoginRequest struct {
		Identifier string `json:"identifier" binding:"required"`
		Password   string `json:"password"   binding:"required"`
	}
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, 400, err.Error())
		return
	}
	var user models.User
	if err := h.DB.Where("LOWER(email) = LOWER(?) OR LOWER(username) = LOWER(?)", req.Identifier, req.Identifier).First(&user).Error; err != nil {
		helpers.Fail(c, 401, "invalid credentials")
		fmt.Println("invalid credentials: ", req.Identifier)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		helpers.Fail(c, 401, "invalid credentials")
		fmt.Println("invalid credentials: ", req.Password)
		return
	}

	access, refresh, err := auth.GenerateTokenPair(&user)
	if err != nil {
		helpers.Fail(c, 500, "could not generate token")
		return
	}

	helpers.OK(c, gin.H{
		"token":         access,
		"refresh_token": refresh,
	})
}

// Refresh exchanges a valid refresh token for a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.Fail(c, 400, "refresh_token is required")
		return
	}

	claims, err := auth.ParseRefreshToken(body.RefreshToken)
	if err != nil {
		helpers.Fail(c, 401, err.Error())
		return
	}

	var user models.User
	if err := h.DB.First(&user, claims.UserID).Error; err != nil {
		helpers.Fail(c, 401, "user not found")
		return
	}
	if !user.IsActive {
		helpers.Fail(c, 403, "account disabled")
		return
	}

	access, newRefresh, err := auth.GenerateTokenPair(&user)
	if err != nil {
		helpers.Fail(c, 500, "could not generate token")
		return
	}

	helpers.OK(c, gin.H{
		"token":         access,
		"refresh_token": newRefresh,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	type RegisterRequest struct {
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"    binding:"required,email"`
		Username string `json:"username" binding:"required"`
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, 400, err.Error())
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}

	newUser := models.User{
		Email:          req.Email,
		Username:       req.Username,
		HashedPassword: string(bytes),
	}

	if err := h.DB.Create(&newUser).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			helpers.Fail(c, 409, "email or username already taken")
			return
		}
		helpers.Fail(c, 500, "could not create user")
		return
	}

	access, refresh, err := auth.GenerateTokenPair(&newUser)
	if err != nil {
		helpers.Fail(c, 500, "user created but could not generate token")
		return
	}

	helpers.OK(c, gin.H{
		"token":         access,
		"refresh_token": refresh,
		"user": gin.H{
			"id":       newUser.ID,
			"email":    newUser.Email,
			"username": newUser.Username,
		},
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	user := auth.CurrentUser(c)

	type Payload struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.Fail(c, 400, err.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.OldPassword)); err != nil {
		helpers.Fail(c, 401, "Old password is not correct")
		return
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		helpers.Fail(c, 500, "Unable to hash the password")
		return
	}
	if err = h.DB.Model(&user).Update("hashed_password", bytes).Error; err != nil {
		helpers.Fail(c, 500, err.Error())
		return
	}
	helpers.OK(c, gin.H{"message": "password updated"})
}
