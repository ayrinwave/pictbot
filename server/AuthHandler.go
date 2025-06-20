package server

import (
	db2 "Golang_Web_App_Bot/db"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"log"
	"net/http"
	"time"
)

type AuthRequest struct {
	InitData string `json:"initData"`
}
type UserProfileData struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	PhotoURL  string
}

func AuthHandler(db *sql.DB, botToken string, tgBot *tgbotapi.BotAPI) gin.HandlerFunc {
	if botToken == "" {
		log.Fatal("❌ Ошибка: TELEGRAM_BOT_TOKEN не был передан в AuthHandler или является пустым.")
	}
	if tgBot == nil {
		log.Fatal("❌ Ошибка: tgBot (экземпляр *tgbotapi.BotAPI) не был передан в AuthHandler. Убедитесь, что вы его инициализировали в main.")
	}

	return func(c *gin.Context) {
		var authReq AuthRequest
		if err := c.ShouldBindJSON(&authReq); err != nil {
			log.Printf("❌ AuthHandler: Ошибка парсинга JSON запроса: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Invalid request format"})
			return
		}

		initDataRaw := authReq.InitData

		if initDataRaw == "" {
			log.Println("❌ initData пуст в JSON-запросе в AuthHandler.")
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "initData is empty"})
			return
		}

		log.Println("▶ AuthHandler: initData получен из JSON (обрезано для лога):", initDataRaw[:min(len(initDataRaw), 100)], "...")

		parsedInitData, err := initdata.Parse(initDataRaw)
		if err != nil {
			log.Printf("❌ AuthHandler: Ошибка парсинга initData: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid initData format"})
			return
		}

		err = initdata.Validate(initDataRaw, botToken, 48*time.Hour)
		if err != nil {
			log.Printf("❌ AuthHandler: Ошибка валидации initData: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": err.Error()})
			return
		}
		log.Println("✅ AuthHandler: initData успешно валидирован.")

		if parsedInitData.User.ID == 0 {
			log.Println("❌ AuthHandler: В валидном initData отсутствует информация о пользователе (User или ID=0).")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "user data missing or invalid in initData"})
			return
		}

		userID := parsedInitData.User.ID
		usernameFromInitData := parsedInitData.User.Username
		firstNameFromInitData := parsedInitData.User.FirstName
		lastNameFromInitData := parsedInitData.User.LastName
		photoURLFromInitData := parsedInitData.User.PhotoURL

		actualProfileData, err := FetchTelegramUserProfile(tgBot, userID)
		if err != nil {
			log.Printf("⚠️ AuthHandler: Не удалось получить актуальные данные профиля пользователя %d из Telegram API: %v. Использую данные из initData.", userID, err)
			actualProfileData = &UserProfileData{
				ID:        userID,
				Username:  usernameFromInitData,
				FirstName: firstNameFromInitData,
				LastName:  lastNameFromInitData,
				PhotoURL:  photoURLFromInitData,
			}
		} else {
			log.Printf("✅ AuthHandler: Актуальные данные профиля пользователя %d получены из Telegram API.", userID)
		}

		dbUser, err := db2.FindOrCreateUser(
			db,
			actualProfileData.ID,
			actualProfileData.Username,
			actualProfileData.FirstName,
			actualProfileData.LastName,
			actualProfileData.PhotoURL,
		)
		if err != nil {
			log.Printf("❌ AuthHandler: Ошибка при FindOrCreateUser в БД: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
			return
		}

		log.Printf("✅ AuthHandler: Пользователь успешно обработан в БД: ID=%d, Username:%s, FirstName:%s, LastName:%s",
			dbUser.TelegramUserID, dbUser.TelegramUsername.String, dbUser.FirstName.String, dbUser.LastName.String)

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"user": gin.H{
				"id":         dbUser.TelegramUserID,
				"username":   dbUser.TelegramUsername.String,
				"first_name": dbUser.FirstName.String,
				"last_name":  dbUser.LastName.String,
				"photo_url":  dbUser.PhotoURL.String,
			},
		})
		log.Println("✅ AuthHandler: Успешный ответ авторизации отправлен.")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func FetchTelegramUserProfile(botAPI *tgbotapi.BotAPI, userID int64) (*UserProfileData, error) {
	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: userID,
		},
	}

	chat, err := botAPI.GetChat(chatConfig)
	if err != nil {
		log.Printf("❌ FetchTelegramUserProfile: Ошибка получения данных чата для пользователя %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка получения данных чата из Telegram: %w", err)
	}

	profileData := &UserProfileData{
		ID:        userID,
		Username:  chat.UserName,
		FirstName: chat.FirstName,
		LastName:  chat.LastName,
		PhotoURL:  "",
	}

	if chat.Photo != nil {
		fileConfig := tgbotapi.FileConfig{
			FileID: chat.Photo.SmallFileID,
		}

		file, err := botAPI.GetFile(fileConfig)
		if err != nil {
			log.Printf("⚠️ FetchTelegramUserProfile: Ошибка получения файла фото для пользователя %d (FileID: %s): %v", userID, chat.Photo.SmallFileID, err)
		} else {
			profileData.PhotoURL = file.Link(botAPI.Token)
			log.Printf("✅ FetchTelegramUserProfile: Получен URL фото пользователя %d: %s", userID, profileData.PhotoURL)
		}
	} else {
		log.Printf("ℹ️ FetchTelegramUserProfile: У пользователя %d нет фото профиля.", userID)
	}

	return profileData, nil
}
