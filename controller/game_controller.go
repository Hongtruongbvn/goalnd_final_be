package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mvc-demo/config"
	"go-mvc-demo/models"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const RAWG_API_KEY = "adcb1c5b37944e2d9b1a9e730d721f44" // <-- Thay bằng API key thật của bạn
// CreateGame godoc
// @Summary Tạo một game mới
// @Description Thêm game mới vào cơ sở dữ liệu
// @Tags Games
// @Accept json
// @Produce json
// @Param game body models.Game true "Game JSON"
// @Success 201 {object} models.Game
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /games [post]
func CreateGame(c *gin.Context) {
	var game models.Game
	if err := c.ShouldBindJSON(&game); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	game.ID = primitive.NewObjectID()

	_, err := config.DB.Collection("games").InsertOne(context.TODO(), game)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, game)
}

// GetGames godoc
// @Summary Lấy danh sách tất cả game
// @Description Trả về danh sách tất cả game từ MongoDB
// @Tags Games
// @Produce json
// @Success 200 {array} models.Game
// @Failure 500 {object} map[string]string
// @Router /games [get]
func GetGames(c *gin.Context) {
	cursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(500, gin.H{"error": "Error retrieving games"})
		return
	}
	defer cursor.Close(context.TODO())

	var games []models.Game
	if err := cursor.All(context.TODO(), &games); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode games"})
		return
	}
	c.JSON(200, games)
}

func GetGameByID(c *gin.Context) {
	idParam := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	var game models.Game
	err = config.DB.Collection("games").FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&game)
	if err != nil {
		c.JSON(404, gin.H{"error": "Game not found"})
		return
	}
	c.JSON(200, game)
}

// DeleteGame godoc
// @Summary Xoá game theo ID
// @Description Xoá game trong MongoDB theo ID
// @Tags Games
// @Param id path string true "Game ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /games/{id} [delete]
func DeleteGame(c *gin.Context) {
	idParam := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = config.DB.Collection("games").DeleteOne(context.TODO(), bson.M{"_id": objectID})
	if err != nil {
		c.JSON(500, gin.H{"error": "Delete failed"})
		return
	}
	c.JSON(200, gin.H{"message": "Game deleted successfully"})
}

// FetchAndSaveGames godoc
// @Summary Import toàn bộ game từ RAWG API (~10,000 game)
// @Description Đồng bộ dữ liệu từ RAWG API
// @Tags Games
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router  /games/fetch  [get]
func FetchAndSaveGames(c *gin.Context) {
	totalPages := 250 // 250 pages * 40 games = 10,000 games
	pageSize := 40
	importedCount := 0

	for page := 1; page <= totalPages; page++ {
		url := fmt.Sprintf("https://api.rawg.io/api/games?key=%s&page_size=%d&page=%d", RAWG_API_KEY, pageSize, page)
		resp, err := http.Get(url)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to fetch from RAWG at page %d", page)})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read response body"})
			return
		}

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to parse JSON"})
			return
		}

		results, ok := result["results"].([]interface{})
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid data format"})
			return
		}

		for _, item := range results {
			gameMap := item.(map[string]interface{})

			rawgID := int(gameMap["id"].(float64))
			name := gameMap["name"].(string)

			image := ""
			if gameMap["background_image"] != nil {
				image = gameMap["background_image"].(string)
			}

			rating := 0.0
			if gameMap["rating"] != nil {
				rating = gameMap["rating"].(float64)
			}

			genresRaw := gameMap["genres"].([]interface{})
			var genres []string
			for _, g := range genresRaw {
				genres = append(genres, g.(map[string]interface{})["name"].(string))
			}

			platformsRaw := gameMap["platforms"].([]interface{})
			var platforms []string
			for _, p := range platformsRaw {
				pMap := p.(map[string]interface{})
				platform := pMap["platform"].(map[string]interface{})
				platforms = append(platforms, platform["name"].(string))
			}

			description := "No description available"

			price := 100 + rand.Intn(900)

			filter := bson.M{"rawg_id": rawgID}
			count, err := config.DB.Collection("games").CountDocuments(context.TODO(), filter)
			if err != nil {
				c.JSON(500, gin.H{"error": "Database error"})
				return
			}
			if count == 0 {
				game := models.Game{
					ID:          primitive.NewObjectID(),
					RawgID:      rawgID,
					Name:        name,
					Description: description,
					ImageURL:    image,
					Genres:      genres,
					Platforms:   platforms,
					Rating:      rating,
					Price:       price,
				}
				_, err := config.DB.Collection("games").InsertOne(context.TODO(), game)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to insert into DB"})
					return
				}
				importedCount++
			}
		}
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("%d games imported", importedCount)})
}

// FetchAndSaveGames100 godoc
// @Summary Import 100 game từ RAWG API
// @Description Chỉ lấy 100 game từ RAWG để kiểm thử
// @Tags Games
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /games/fetch-games100 [get]
func FetchAndSaveGames100(c *gin.Context) {
	const totalGames = 100
	const pageSize = 40

	importedCount := 0
	fetchedCount := 0

	totalPages := (totalGames + pageSize - 1) / pageSize // = 3 pages

	for page := 1; page <= totalPages; page++ {
		url := fmt.Sprintf("https://api.rawg.io/api/games?key=%s&page_size=%d&page=%d", RAWG_API_KEY, pageSize, page)
		resp, err := http.Get(url)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to fetch from RAWG at page %d: %v", page, err)})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read response body"})
			return
		}

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to parse JSON"})
			return
		}

		resultsRaw, ok := result["results"].([]interface{})
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid data format in results"})
			return
		}

		for _, item := range resultsRaw {
			if importedCount >= totalGames {
				break
			}

			gameMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			idVal, ok := gameMap["id"].(float64)
			if !ok {
				continue
			}
			rawgID := int(idVal)

			nameVal, ok := gameMap["name"].(string)
			if !ok {
				continue
			}
			name := nameVal

			image := ""
			if imgVal, ok := gameMap["background_image"].(string); ok {
				image = imgVal
			}

			rating := 0.0
			if ratingVal, ok := gameMap["rating"].(float64); ok {
				rating = ratingVal
			}

			genres := []string{}
			if genresRaw, ok := gameMap["genres"].([]interface{}); ok {
				for _, g := range genresRaw {
					if gMap, ok := g.(map[string]interface{}); ok {
						if gName, ok := gMap["name"].(string); ok {
							genres = append(genres, gName)
						}
					}
				}
			}

			platforms := []string{}
			if platformsRaw, ok := gameMap["platforms"].([]interface{}); ok {
				for _, p := range platformsRaw {
					if pMap, ok := p.(map[string]interface{}); ok {
						if platform, ok := pMap["platform"].(map[string]interface{}); ok {
							if platformName, ok := platform["name"].(string); ok {
								platforms = append(platforms, platformName)
							}
						}
					}
				}
			}

			description := "No description available" // API chưa lấy mô tả chi tiết, nếu cần có thể fetch thêm

			price := 100 + rand.Intn(900)

			filter := bson.M{"rawg_id": rawgID}
			count, err := config.DB.Collection("games").CountDocuments(context.TODO(), filter)
			if err != nil {
				c.JSON(500, gin.H{"error": "Database error"})
				return
			}
			if count == 0 {
				game := models.Game{
					ID:          primitive.NewObjectID(),
					RawgID:      rawgID,
					Name:        name,
					Description: description,
					ImageURL:    image,
					Genres:      genres,
					Platforms:   platforms,
					Rating:      rating,
					Price:       price,
				}
				_, err := config.DB.Collection("games").InsertOne(context.TODO(), game)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to insert into DB"})
					return
				}
				importedCount++
			}

			fetchedCount++
		}
		if importedCount >= totalGames {
			break
		}
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("%d games imported out of %d fetched", importedCount, fetchedCount)})
}

// FetchGamesByPage godoc
// @Summary Lấy danh sách game theo phân trang
// @Description Trả về 100 game mỗi trang, với tổng số trang
// @Tags Games
// @Produce json
// @Param page query int false "Page number" default(1)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /games/fetch-games [get]
func FetchGamesByPage(c *gin.Context) {
	pageStr := c.Query("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(400, gin.H{"error": "Invalid page"})
		return
	}

	pageSize := 100
	skip := (page - 1) * pageSize

	cursor, err := config.DB.Collection("games").Find(
		context.TODO(),
		bson.M{},
		options.Find().SetLimit(int64(pageSize)).SetSkip(int64(skip)),
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error retrieving games"})
		return
	}
	defer cursor.Close(context.TODO())

	var games []models.Game
	if err := cursor.All(context.TODO(), &games); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode games"})
		return
	}

	totalGames, err := config.DB.Collection("games").CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(500, gin.H{"error": "Error counting games"})
		return
	}

	totalPages := int(math.Ceil(float64(totalGames) / float64(pageSize)))

	c.JSON(200, gin.H{
		"games":      games,
		"page":       page,
		"totalPages": totalPages,
		"totalGames": totalGames,
	})
}

// GetPurchasedGames godoc
// @Summary Lấy danh sách game đã mua
// @Description Trả về danh sách các game mà người dùng đã mua
// @Tags Games
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/my-purchases [get]
func GetPurchasedGames(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	cursor, err := config.DB.Collection("purchases").Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch purchases"})
		return
	}
	defer cursor.Close(context.TODO())

	var purchases []models.Purchase
	if err := cursor.All(context.TODO(), &purchases); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode purchases"})
		return
	}

	var gameIDs []primitive.ObjectID
	for _, p := range purchases {
		gameIDs = append(gameIDs, p.GameID)
	}

	gameCursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{"_id": bson.M{"$in": gameIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}
	defer gameCursor.Close(context.TODO())

	gameMap := make(map[primitive.ObjectID]models.Game)
	for gameCursor.Next(context.TODO()) {
		var game models.Game
		if err := gameCursor.Decode(&game); err == nil {
			gameMap[game.ID] = game
		}
	}

	var result []gin.H
	for _, p := range purchases {
		game := gameMap[p.GameID]
		result = append(result, gin.H{
			"id":          p.ID.Hex(),
			"price":       p.Price,
			"purchase_at": p.PurchaseAt,
			"game": gin.H{
				"id":        game.ID.Hex(),
				"name":      game.Name,
				"image_url": game.ImageURL,
				"price":     game.Price,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"purchases": result})
}

// GetRentedGames godoc
// @Summary Lấy danh sách game đã thuê
// @Description Trả về danh sách các game mà người dùng đã thuê
// @Tags Games
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/my-rentals [get]
func GetRentedGames(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	cursor, err := config.DB.Collection("rentals").Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rentals"})
		return
	}
	defer cursor.Close(context.TODO())

	var rentals []models.Rental
	if err := cursor.All(context.TODO(), &rentals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode rentals"})
		return
	}

	var gameIDs []primitive.ObjectID
	for _, r := range rentals {
		gameIDs = append(gameIDs, r.GameID)
	}

	gameCursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{"_id": bson.M{"$in": gameIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}
	defer gameCursor.Close(context.TODO())

	gameMap := make(map[primitive.ObjectID]models.Game)
	for gameCursor.Next(context.TODO()) {
		var game models.Game
		if err := gameCursor.Decode(&game); err == nil {
			gameMap[game.ID] = game
		}
	}

	var results []gin.H
	for _, rental := range rentals {
		game := gameMap[rental.GameID]
		results = append(results, gin.H{
			"id":        rental.ID.Hex(),
			"rent_at":   rental.RentAt,
			"expire_at": rental.ExpireAt,
			"status":    rental.Status,
			"game": gin.H{
				"id":        game.ID.Hex(),
				"name":      game.Name,
				"image_url": game.ImageURL,
				"price":     game.Price,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"rentals": results})
}
