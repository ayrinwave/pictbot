package server // Или ваш пакет

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

// Структура для получения JSON-тела запроса
type AuthRequest struct {
	InitData string `json:"initData"`
}
type UserProfileData struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	PhotoURL  string // URL к аватару, полученный от Telegram
}

func AuthHandler(db *sql.DB, botToken string, tgBot *tgbotapi.BotAPI) gin.HandlerFunc { // <-- ДОБАВЛЕНО tgBot
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

		log.Println("▶ AuthHandler: initData получен из JSON (обрезано для лога):", initDataRaw[:min(len(initDataRaw), 100)], "...") // Обрезаем длинный лог

		parsedInitData, err := initdata.Parse(initDataRaw)
		if err != nil {
			log.Printf("❌ AuthHandler: Ошибка парсинга initData: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid initData format"})
			return
		}

		// Валидация initData
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

		// Данные из initData (могут быть не самыми свежими, особенно photo_url)
		userID := parsedInitData.User.ID
		usernameFromInitData := parsedInitData.User.Username
		firstNameFromInitData := parsedInitData.User.FirstName
		lastNameFromInitData := parsedInitData.User.LastName
		photoURLFromInitData := parsedInitData.User.PhotoURL

		// === НОВОЕ: Запрашиваем актуальные данные профиля через Telegram Bot API ===
		// Используем нашу новую функцию FetchTelegramUserProfile
		actualProfileData, err := FetchTelegramUserProfile(tgBot, userID) // <-- ВЫЗЫВАЕМ НОВУЮ ФУНКЦИЮ
		if err != nil {
			log.Printf("⚠️ AuthHandler: Не удалось получить актуальные данные профиля пользователя %d из Telegram API: %v. Использую данные из initData.", userID, err)
			// Если произошла ошибка, используем данные из initData как запасной вариант
			actualProfileData = &UserProfileData{ // Создаем временную структуру с данными из initData
				ID:        userID,
				Username:  usernameFromInitData,
				FirstName: firstNameFromInitData,
				LastName:  lastNameFromInitData,
				PhotoURL:  photoURLFromInitData,
			}
		} else {
			// Если данные получены успешно, обновляем лог
			log.Printf("✅ AuthHandler: Актуальные данные профиля пользователя %d получены из Telegram API.", userID)
		}

		// --- Передаем актуальные данные пользователя в FindOrCreateUser ---
		// Используем данные из actualProfileData, которые теперь могут быть свежее
		dbUser, err := db2.FindOrCreateUser(
			db,
			actualProfileData.ID,
			actualProfileData.Username,
			actualProfileData.FirstName,
			actualProfileData.LastName,
			actualProfileData.PhotoURL, // Используем PhotoURL, полученный из Bot API (или из initData, если API не доступно)
		)
		if err != nil {
			log.Printf("❌ AuthHandler: Ошибка при FindOrCreateUser в БД: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
			return
		}

		log.Printf("✅ AuthHandler: Пользователь успешно обработан в БД: ID=%d, Username:%s, FirstName:%s, LastName:%s",
			dbUser.TelegramUserID, dbUser.TelegramUsername.String, dbUser.FirstName.String, dbUser.LastName.String)

		// --- Возвращаем полные данные пользователя на фронтенд ---
		// Важно: здесь мы возвращаем данные *из БД*, а не из actualProfileData напрямую,
		// чтобы быть уверенными, что это именно то, что было сохранено.
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

// min - вспомогательная функция для обрезки строки для лога (можно убрать, если не нужна)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func FetchTelegramUserProfile(botAPI *tgbotapi.BotAPI, userID int64) (*UserProfileData, error) {
	// Создаем конфигурацию для запроса информации о чате/пользователе.
	// Для обычного пользователя, ChatID равен UserID.
	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: userID,
		},
	}

	// Отправляем запрос GetChat, чтобы получить общую информацию о пользователе.
	chat, err := botAPI.GetChat(chatConfig)
	if err != nil {
		log.Printf("❌ FetchTelegramUserProfile: Ошибка получения данных чата для пользователя %d: %v", userID, err)
		return nil, fmt.Errorf("ошибка получения данных чата из Telegram: %w", err)
	}

	// Инициализируем структуру для возврата данных профиля.
	profileData := &UserProfileData{
		ID:        userID,
		Username:  chat.UserName,  // Может быть пустым, если у пользователя нет юзернейма
		FirstName: chat.FirstName, // Имя пользователя
		LastName:  chat.LastName,  // Фамилия пользователя, может быть пустым
		PhotoURL:  "",             // Изначально пустая строка для URL фото
	}

	// Проверяем, есть ли у пользователя фото профиля.
	if chat.Photo != nil {
		// Telegram API предоставляет FileID, а не прямой URL для фото профиля чата.
		// Нам нужно получить прямую ссылку через метод getFile.
		// Используем SmallFileID или BigFileID в зависимости от нужного размера.
		// SmallFileID обычно быстрее, BigFileID - выше качество.
		fileConfig := tgbotapi.FileConfig{
			FileID: chat.Photo.SmallFileID, // Вы можете попробовать BigFileID для лучшего качества
		}

		file, err := botAPI.GetFile(fileConfig)
		if err != nil {
			log.Printf("⚠️ FetchTelegramUserProfile: Ошибка получения файла фото для пользователя %d (FileID: %s): %v", userID, chat.Photo.SmallFileID, err)
			// Не возвращаем ошибку отсюда, если не удалось получить фото.
			// Просто profileData.PhotoURL останется пустым, и будет использована заглушка.
		} else {
			// Метод Link() генерирует прямую ссылку на файл на серверах Telegram.
			profileData.PhotoURL = file.Link(botAPI.Token)
			log.Printf("✅ FetchTelegramUserProfile: Получен URL фото пользователя %d: %s", userID, profileData.PhotoURL)
		}
	} else {
		log.Printf("ℹ️ FetchTelegramUserProfile: У пользователя %d нет фото профиля.", userID)
	}

	return profileData, nil
}
