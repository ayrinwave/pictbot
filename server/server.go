package server

import (
	bot "Golang_Web_App_Bot/bot"
	"database/sql"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const UPLOADS_BASE_PATH_FOR_WRITING = "D:/Golang_Web_App_Bot_Test/uploads"

// StartServer запускает веб-сервер Gin
func StartServer(db *sql.DB, botToken string) {

	router := gin.Default()
	router.MaxMultipartMemory = 100 << 20 // 100MB
	router.LoadHTMLGlob("templates/*.html")

	router.Static("/static", "./templates/static")
	router.Static("/uploads", "./uploads")

	tgBotAPI, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации Telegram Bot API: %v", err)
	}
	tgBotAPI.Debug = false
	log.Printf("✅ Бот авторизован как %s", tgBotAPI.Self.UserName)
	router.POST("/auth", AuthHandler(db, botToken, tgBotAPI)) // Убедитесь, что AuthHandler находится в пакете bot

	// НОВЫЙ МАРШРУТ: API для получения профиля пользователя по ID
	router.GET("/api/user_profile/:userID", func(c *gin.Context) {
		userIDStr := c.Param("userID")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Неверный ID пользователя"})
			return
		}

		user, err := bot.GetUserProfileByID(db, userID) // Вам нужно реализовать эту функцию
		if err != nil {
			log.Printf("❌ Ошибка получения профиля пользователя ID %d: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не удалось загрузить профиль пользователя"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"user": user,
		})
	})

	// НОВЫЙ МАРШРУТ: API для получения галерей конкретного пользователя по ID
	router.GET("/api/user_galleries/:userID", func(c *gin.Context) {
		userIDStr := c.Param("userID")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Неверный ID пользователя"})
			return
		}

		searchQuery := c.Query("q")

		// --- Извлечение viewerUserID для избранного статуса ---
		var viewerUserID int64 = 0
		initDataRaw := c.GetHeader("X-Telegram-Init-Data")
		if initDataRaw != "" {
			if validateErr := initdata.Validate(initDataRaw, botToken, 24*time.Hour); validateErr != nil {
				log.Printf("⚠️ /api/user_galleries: Ошибка валидации initData: %v. Продолжаем без viewerUserID.", validateErr)
			} else {
				if parsedData, parseErr := initdata.Parse(initDataRaw); parseErr != nil {
					log.Printf("⚠️ /api/user_galleries: Ошибка парсинга валидного initData: %v. Продолжаем без viewerUserID.", parseErr)
				} else if parsedData.User.ID != 0 { // <-- ИЗМЕНЕНО ЗДЕСЬ
					viewerUserID = parsedData.User.ID
				}
			}
		}
		// --- Конец извлечения viewerUserID ---

		// Вам нужно реализовать GetGalleriesByUserID в пакете bot,
		// которая теперь должна принимать viewerUserID
		galleries, fetchErr := bot.GetGalleriesByUserID(db, userID, searchQuery, viewerUserID) // <-- viewerUserID передан
		if fetchErr != nil {
			log.Printf("❌ Ошибка получения галерей для пользователя ID %d: %v", userID, fetchErr)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не удалось загрузить галереи пользователя"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":        true,
			"galleries": galleries,
		})
	})

	router.GET("/api/gallery_images/:galleryID", func(c *gin.Context) {
		galleryIDStr := c.Param("galleryID")
		galleryID, err := strconv.ParseInt(galleryIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Неверный ID галереи"})
			return
		}

		// `GetGalleryImages` теперь возвращает пути типа "gallery_images/123/image_hash_full.jpg"
		imageDBPaths, err := bot.GetGalleryImages(db, galleryID) // bot.GetGalleryImages - это ваша функция
		if err != nil {
			log.Printf("❌ Ошибка получения изображений для галереи ID %d: %v", galleryID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не удалось загрузить изображения галереи"})
			return
		}

		var fullImageURLs []string
		for _, dbPath := range imageDBPaths {
			// Формируем URL для фронтенда, используя новый безопасный префикс
			// Этот префикс должен соответствовать роуту, который вы создадите для отдачи файлов
			fullImageURLs = append(fullImageURLs, "/secured_gallery_images/"+dbPath) // <-- Изменено
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"images": fullImageURLs,
		})
	})
	// Добавляем новый обработчик для безопасной отдачи файлов
	// Он будет принимать URL вида /secured_gallery_images/gallery_images/123/image.jpg
	router.GET("/secured_gallery_images/*filepath", func(c *gin.Context) {
		requestedPath := strings.TrimPrefix(c.Param("filepath"), "/") // Удаляем начальный слэш, если есть

		// **Важная проверка безопасности:** предотвращение обхода директорий.
		cleanPath := filepath.Clean(requestedPath)

		if strings.HasPrefix(cleanPath, "..") {
			log.Printf("❌ Попытка обхода директории: %s", requestedPath)
			c.String(http.StatusBadRequest, "Неверный путь к файлу")
			return
		}

		// Формируем полный физический путь к файлу на сервере.
		// 'uploads' - это базовая директория, где лежат все ваши загрузки.
		fullFilePath := filepath.Join(UPLOADS_BASE_PATH_FOR_WRITING, cleanPath) // UPLOADS_BASE_PATH_FOR_WRITING

		if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
			log.Printf("❌ Запрошенный файл не найден: %s (полный путь: %s)", requestedPath, fullFilePath)
			c.String(http.StatusNotFound, "Файл не найден")
			return
		}

		c.File(fullFilePath)
	})

	authorized := router.Group("/api")
	authorized.Use(bot.AuthMiddleware(db, botToken))
	{
		authorized.POST("/add_gallery", bot.AddGalleryHandler(db))
		authorized.POST("/my_galleries_data", bot.GetMyGalleriesAPIHandler(db))
		authorized.DELETE("/delete_gallery/:galleryName", bot.DeleteGalleryHandler(db))
		authorized.GET("/subscription/status/:targetUserID", bot.CheckSubscriptionStatusHandler(db))
		authorized.POST("/subscription/:targetUserID", bot.SubscribeHandler(db))
		authorized.DELETE("/subscription/:targetUserID", bot.UnsubscribeHandler(db))
		authorized.GET("/my_subscriptions", bot.GetSubscribedUsersHandler(db))
		authorized.GET("/my_favorite_galleries", bot.GetFavoriteGalleriesHandler(db))
		authorized.POST("/favorites/:galleryID", bot.AddFavoriteHandler(db)) // Добавить в избранное
		authorized.DELETE("/favorites/:galleryID", bot.RemoveFavoriteHandler(db))

	}
	// Главная страница
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	// НОВЫЙ МАРШРУТ: Страница галерей пользователя
	router.GET("/user_galleries", func(c *gin.Context) {
		c.HTML(http.StatusOK, "user_galleries.html", gin.H{})
	})

	router.GET("/create_gallery", func(c *gin.Context) {
		c.HTML(http.StatusOK, "create_gallery.html", gin.H{})
	})

	router.GET("/my_galleries", func(c *gin.Context) {
		c.HTML(http.StatusOK, "view_gallery.html", gin.H{})
	})

	router.GET("/view_gallery", func(c *gin.Context) {
		c.HTML(http.StatusOK, "view_gallery.html", gin.H{})
	})
	// НОВЫЙ МАРШРУТ для страницы "Мои подписки"
	router.GET("/my_subscriptions", func(c *gin.Context) {
		c.HTML(http.StatusOK, "subscribed-users.html", nil)
	})
	// НОВЫЙ МАРШРУТ для страницы "Избранные галереи"
	router.GET("/favorite_galleries", func(c *gin.Context) {
		c.HTML(http.StatusOK, "favorite_galleries.html", nil)
	})
	// API для получения всех галерей (с пагинацией и поиском)
	router.GET("/api/galleries", GetGalleriesHandler(db, botToken)) // <--- УБЕДИТЕСЬ, ЧТО ЭТА СТРОКА ЕСТЬ И НЕ ЗАКОММЕНТИРОВАНА

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Веб-сервер запущен на http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Ошибка при запуске веб-сервера: %v", err)
	}
}

func GetGalleriesHandler(db *sql.DB, botToken string) gin.HandlerFunc {
	if botToken == "" {
		log.Fatal("❌ Ошибка: botToken не был передан в GetGalleriesHandler или является пустым.")
	}

	return func(c *gin.Context) {
		searchQuery := c.Query("q")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 20
			log.Printf("⚠️ Неверное значение limit '%s', использую по умолчанию %d", limitStr, limit)
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
			log.Printf("⚠️ Неверное значение offset '%s', использую по умолчанию %d", offsetStr, offset)
		}

		// --- Извлечение user ID из initData ---
		var viewerUserID int64 = 0 // Значение по умолчанию, если не авторизован
		initDataRaw := c.GetHeader("X-Telegram-Init-Data")
		if initDataRaw != "" {
			// Валидация initData с использованием botToken
			err = initdata.Validate(initDataRaw, botToken, 24*time.Hour) // 24 часа - пример, настройте по необходимости
			if err != nil {
				log.Printf("⚠️ GetGalleriesHandler: Ошибка валидации initData: %v. Продолжаем без user ID.", err)

			} else {
				parsedInitData, parseErr := initdata.Parse(initDataRaw)
				if parseErr != nil {
					log.Printf("⚠️ GetGalleriesHandler: Ошибка парсинга валидного initData: %v. Продолжаем без user ID.", parseErr)
				} else if parsedInitData.User.ID != 0 { // <-- ИЗМЕНЕНО: Проверяем User.ID != 0
					viewerUserID = parsedInitData.User.ID
					log.Printf("✅ GetGalleriesHandler: Получен viewerUserID: %d из initData.", viewerUserID)
				} else {
					log.Println("⚠️ GetGalleriesHandler: Валидный initData не содержит данных пользователя (User.ID=0). Продолжаем без user ID.")
				}
			}
		} else {
			log.Println("ℹ️ GetGalleriesHandler: Заголовок X-Telegram-Init-Data отсутствует. Галереи будут загружены без учета избранного статуса.")
		}
		// --- Конец извлечения user ID ---

		var galleries []bot.Gallery
		var fetchErr error

		if searchQuery != "" {
			// ИЗМЕНЕНО: Передаем viewerUserID в функцию GetGalleriesByTag
			galleries, fetchErr = bot.GetGalleriesByTag(db, searchQuery, viewerUserID, limit, offset)
		} else {
			// ИЗМЕНЕНО: Передаем viewerUserID в функцию GetAllGalleries
			galleries, fetchErr = bot.GetAllGalleries(db, viewerUserID, limit, offset)
		}

		if fetchErr != nil {
			log.Printf("❌ Ошибка получения галерей для API: %v", fetchErr)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to fetch galleries"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":        true,
			"galleries": galleries,
		})
	}
}
