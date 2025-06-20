package bot

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// checkSubscriptionStatusHandler проверяет, подписан ли текущий пользователь на целевого
func CheckSubscriptionStatusHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic in checkSubscriptionStatusHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Internal server error"})
			}
		}()

		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ checkSubscriptionStatusHandler: User not authenticated or userID not set")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "User not authenticated"})
			return
		}
		subscriberID, ok := sessionUserID.(int64)
		if !ok || subscriberID <= 0 {
			log.Printf("❌ checkSubscriptionStatusHandler: Invalid userID format or value: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Invalid user ID"})
			return
		}

		targetUserIDStr := c.Param("targetUserID")
		targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
		if err != nil {
			log.Printf("❌ checkSubscriptionStatusHandler: Invalid target user ID in URL: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Invalid target user ID"})
			return
		}

		if subscriberID == targetUserID {
			// Пользователь просматривает свой профиль, не может быть подписан сам на себя
			c.JSON(http.StatusOK, gin.H{"ok": true, "isSubscribed": false})
			return
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE subscriber_id = $1 AND target_user_id = $2", subscriberID, targetUserID).Scan(&count)
		if err != nil {
			log.Printf("❌ checkSubscriptionStatusHandler: Error checking subscription status for subscriber %d to target %d: %v", subscriberID, targetUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to check subscription status"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "isSubscribed": count > 0})
	}
}

// subscribeHandler добавляет подписку
func SubscribeHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic in subscribeHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Internal server error"})
			}
		}()

		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ subscribeHandler: User not authenticated or userID not set")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "User not authenticated"})
			return
		}
		subscriberID, ok := sessionUserID.(int64)
		if !ok || subscriberID <= 0 {
			log.Printf("❌ subscribeHandler: Invalid userID format or value: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Invalid user ID"})
			return
		}

		targetUserIDStr := c.Param("targetUserID")
		targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
		if err != nil {
			log.Printf("❌ subscribeHandler: Invalid target user ID in URL: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Invalid target user ID"})
			return
		}

		if subscriberID == targetUserID {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Cannot subscribe to yourself"})
			return
		}

		// Проверяем, существует ли целевой пользователь
		var userExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE telegram_user_id = $1)", targetUserID).Scan(&userExists)
		if err != nil {
			log.Printf("❌ subscribeHandler: Error checking existence of target user %d: %v", targetUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to check target user existence"})
			return
		}
		if !userExists {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "Target user not found"})
			return
		}

		_, err = db.Exec("INSERT INTO subscriptions (subscriber_id, target_user_id, created_at) VALUES ($1, $2, $3)",
			subscriberID, targetUserID, time.Now())
		if err != nil {
			// Если подписка уже существует (primary key violation)
			if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "UNIQUE constraint failed") { // Для PostgreSQL/SQLite
				log.Printf("⚠️ subscribeHandler: User %d is already subscribed to %d", subscriberID, targetUserID)
				c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "Already subscribed"})
				return
			}
			log.Printf("❌ subscribeHandler: Error adding subscription for subscriber %d to target %d: %v", subscriberID, targetUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to subscribe"})
			return
		}

		log.Printf("✅ subscribeHandler: User %d subscribed to %d", subscriberID, targetUserID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// unsubscribeHandler удаляет подписку
func UnsubscribeHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic in unsubscribeHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Internal server error"})
			}
		}()

		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ unsubscribeHandler: User not authenticated or userID not set")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "User not authenticated"})
			return
		}
		subscriberID, ok := sessionUserID.(int64)
		if !ok || subscriberID <= 0 {
			log.Printf("❌ unsubscribeHandler: Invalid userID format or value: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Invalid user ID"})
			return
		}

		targetUserIDStr := c.Param("targetUserID")
		targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
		if err != nil {
			log.Printf("❌ unsubscribeHandler: Invalid target user ID in URL: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Invalid target user ID"})
			return
		}

		if subscriberID == targetUserID {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Cannot unsubscribe from yourself"})
			return
		}

		result, err := db.Exec("DELETE FROM subscriptions WHERE subscriber_id = $1 AND target_user_id = $2", subscriberID, targetUserID)
		if err != nil {
			log.Printf("❌ unsubscribeHandler: Error deleting subscription for subscriber %d from target %d: %v", subscriberID, targetUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to unsubscribe"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("❌ unsubscribeHandler: Error getting rows affected after deletion: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to unsubscribe"})
			return
		}

		if rowsAffected == 0 {
			log.Printf("⚠️ unsubscribeHandler: Subscription for subscriber %d to target %d not found.", subscriberID, targetUserID)
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "Subscription not found"})
			return
		}

		log.Printf("✅ unsubscribeHandler: User %d unsubscribed from %d", subscriberID, targetUserID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetSubscribedUsersHandler обрабатывает запрос на получение списка пользователей,
// на которых подписан текущий авторизованный пользователь.
func GetSubscribedUsersHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic in GetSubscribedUsersHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Internal server error"})
			}
		}()

		userID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ GetSubscribedUsersHandler: User ID not found in context.")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Authentication required"})
			return
		}
		currentUserID, ok := userID.(int64)
		if !ok || currentUserID <= 0 {
			log.Printf("❌ GetSubscribedUsersHandler: Invalid user ID format in context: %v", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Invalid user ID"})
			return
		}

		var subscribedUsers []SubscribedUserProfile
		// Запрос к БД для получения пользователей, на которых подписан currentUserID
		rows, err := db.Query(`
            SELECT
                u.telegram_user_id,
                u.telegram_username,
                u.first_name,
                u.last_name,
                u.photo_url
            FROM
                subscriptions s
            JOIN
                users u ON s.target_user_id = u.telegram_user_id
            WHERE
                s.subscriber_id = $1
            ORDER BY
                u.first_name ASC, u.telegram_username ASC;
        `, currentUserID)
		if err != nil {
			log.Printf("❌ GetSubscribedUsersHandler: Error querying subscribed users for %d: %v", currentUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to retrieve subscribed users"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var user SubscribedUserProfile
			err := rows.Scan(
				&user.TelegramUserID,
				&user.TelegramUsername,
				&user.FirstName,
				&user.LastName,
				&user.PhotoURL,
			)
			if err != nil {
				log.Printf("❌ GetSubscribedUsersHandler: Error scanning subscribed user row: %v", err)
				// Продолжаем, чтобы не прерывать весь список из-за одной ошибки,
				// но в реальном приложении можно рассмотреть более строгую обработку
				continue
			}
			subscribedUsers = append(subscribedUsers, user)
		}

		if err = rows.Err(); err != nil {
			log.Printf("❌ GetSubscribedUsersHandler: Error after iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Error processing subscribed users data"})
			return
		}

		log.Printf("✅ GetSubscribedUsersHandler: Retrieved %d subscribed users for user %d.", len(subscribedUsers), currentUserID)
		c.JSON(http.StatusOK, gin.H{"ok": true, "users": subscribedUsers})
	}
}

// Вспомогательная функция для парсинга тегов из строки БД в срез строк
func ParseTags(tagString sql.NullString) []string {
	if !tagString.Valid || tagString.String == "" {
		return []string{}
	}
	// Предполагаем, что теги хранятся как строка, разделенная запятыми (например, "tag1,tag2,tag3")
	tags := strings.Split(tagString.String, ",")
	// Очищаем пробелы вокруг тегов
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}
	return tags
}

// GetFavoriteGalleriesHandler обрабатывает запрос на получение списка избранных галерей
func GetFavoriteGalleriesHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic in GetFavoriteGalleriesHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Internal server error"})
			}
		}()

		userID, exists := c.Get("userID") // Получаем userID из контекста, установленного AuthMiddleware
		if !exists {
			log.Println("❌ GetFavoriteGalleriesHandler: User ID not found in context (AuthMiddleware missing or failed).")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Authentication required"})
			return
		}
		currentUserID, ok := userID.(int64)
		if !ok || currentUserID <= 0 {
			log.Printf("❌ GetFavoriteGalleriesHandler: Invalid user ID format in context: %v", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Invalid user ID"})
			return
		}

		var favoriteGalleries []GalleryFullDetail

		// ИЗМЕНЕННЫЙ ЗАПРОС К БД: ДОБАВЛЕН JOIN С ТАБЛИЦЕЙ USERS
		rows, err := db.Query(`
            SELECT
                g.id,
                g.name,
                g.user_id,
                COALESCE(
                    (SELECT i.preview_image_path FROM gallery_images i WHERE i.gallery_id = g.id ORDER BY i.id LIMIT 1),
                    ''
                ) AS preview_url,
                (SELECT COUNT(*) FROM gallery_images WHERE gallery_id = g.id) AS image_count,
                t.tags_string,
                g.created_at,
                -- Добавляем поля пользователя-создателя
                u.telegram_user_id,
                u.telegram_username,
                u.first_name,
                u.last_name,
                u.photo_url
            FROM
                galleries g
            JOIN
                favorites f ON g.id = f.gallery_id
            LEFT JOIN
                (SELECT gallery_id, STRING_AGG(tag, ', ') AS tags_string FROM tags GROUP BY gallery_id) t
                ON g.id = t.gallery_id
            JOIN -- ИЗМЕНЕНО: Добавляем JOIN с таблицей users
                users u ON g.user_id = u.telegram_user_id
            WHERE
                f.user_id = $1
            ORDER BY
                g.created_at DESC;
        `, currentUserID)
		if err != nil {
			log.Printf("❌ GetFavoriteGalleriesHandler: Error querying favorite galleries for user %d: %v", currentUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to retrieve favorite galleries"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var g GalleryFullDetail
			var tagsString sql.NullString
			var previewURL sql.NullString
			var creatorUsername sql.NullString
			var creatorFirstName sql.NullString
			var creatorLastName sql.NullString
			var creatorPhotoURL sql.NullString

			err := rows.Scan(
				&g.ID,
				&g.Name,
				&g.UserID,
				&previewURL,
				&g.ImageCount,
				&tagsString,
				&g.CreatedAt,
				&g.CreatorID,
				&creatorUsername,
				&creatorFirstName,
				&creatorLastName,
				&creatorPhotoURL,
			)
			if err != nil {
				log.Printf("❌ GetFavoriteGalleriesHandler: Error scanning favorite gallery row: %v", err)
				continue
			}

			g.PreviewURL = previewURL.String
			g.Tags = ParseTags(tagsString)
			g.CreatorUsername = creatorUsername
			g.CreatorFirstName = creatorFirstName
			g.CreatorLastName = creatorLastName
			g.CreatorPhotoURL = creatorPhotoURL

			favoriteGalleries = append(favoriteGalleries, g) // <<< ВОТ ЭТА СТРОКА ОШИБЛАСЬ
		}

		if err = rows.Err(); err != nil {
			log.Printf("❌ GetFavoriteGalleriesHandler: Error after iterating rows: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Error processing favorite galleries data"})
			return
		}

		log.Printf("✅ GetFavoriteGalleriesHandler: Retrieved %d favorite galleries for user %d.", len(favoriteGalleries), currentUserID)
		c.JSON(http.StatusOK, gin.H{"ok": true, "galleries": favoriteGalleries})
	}
}

// AddFavoriteHandler (POST /api/favorites/:galleryID)
func AddFavoriteHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUserID := c.GetInt64("userID") // Получаем UserID из AuthMiddleware
		if currentUserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован."})
			return
		}

		galleryIDStr := c.Param("galleryID")
		galleryID, err := strconv.ParseInt(galleryIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Неверный ID галереи."})
			return
		}

		_, err = db.Exec("INSERT INTO favorites (user_id, gallery_id) VALUES ($1, $2) ON CONFLICT (user_id, gallery_id) DO NOTHING",
			currentUserID, galleryID)
		if err != nil {
			log.Printf("Error adding favorite: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// RemoveFavoriteHandler (DELETE /api/favorites/:galleryID)
func RemoveFavoriteHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Handling DELETE /api/favorites/:galleryID request...") // Добавьте это
		currentUserID := c.GetInt64("userID")                               // Получаем UserID из AuthMiddleware
		log.Printf("Current User ID from context: %d", currentUserID)       // Добавьте это

		if currentUserID == 0 {
			log.Println("Authentication failed: UserID is 0.") // Добавьте это
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован."})
			return
		}

		galleryIDStr := c.Param("galleryID")
		log.Printf("Attempting to remove favorite for gallery ID: %s", galleryIDStr) // Добавьте это
		galleryID, err := strconv.ParseInt(galleryIDStr, 10, 64)
		if err != nil {
			log.Printf("Invalid gallery ID format: %s, error: %v", galleryIDStr, err) // Добавьте это
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Неверный ID галереи."})
			return
		}

		_, err = db.Exec("DELETE FROM favorites WHERE user_id = $1 AND gallery_id = $2",
			currentUserID, galleryID)
		if err != nil {
			log.Printf("Error removing favorite from DB: %v", err) // Добавьте это
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не удалось удалить из избранного."})
			return
		}

		log.Println("Gallery successfully removed from favorites.") // Добавьте это
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Галерея удалена из избранного."})
	}
}
