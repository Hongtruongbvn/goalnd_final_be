package controllers

import (
	"context"
	"go-mvc-demo/config"
	"go-mvc-demo/models"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("SECRET_KEY")

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Register godoc
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body models.User true "User registration info"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/register [post]
func Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 12)

	user := models.User{
		ID:          primitive.NewObjectID(),
		Name:        input.Name,
		Email:       input.Email,
		Password:    string(hashedPassword),
		CoinBalance: 1000, // khởi tạo coin
		Role:        "user",
	}

	_, err := config.DB.Collection("users").InsertOne(context.TODO(), user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Cannot create user"})
		return
	}

	c.JSON(201, gin.H{"message": "User created"})
}

// Login godoc
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body map[string]string true "Email and Password"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	var user models.User
	err := config.DB.Collection("users").FindOne(context.TODO(), bson.M{"email": input.Email}).Decode(&user)
	if err != nil {
		c.JSON(401, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(401, gin.H{"error": "Wrong password"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(200, gin.H{"token": tokenString})
}
