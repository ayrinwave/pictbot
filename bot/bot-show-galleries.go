package bot

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type AuthRequest struct {
	InitData string `json:"initData"`
}

const UPLOADS_PHYSICAL_BASE_PATH = "D:/Golang_Web_App_Bot_Test/uploads/"

func isImageFile(filename string) bool {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		log.Printf("⚠️ Файл %s не является изображением (расширение: %s)", filename, ext)
		return false
	}
}

func GetUserGalleries(db *sql.DB, userID int64) ([]Gallery, error) {
	var galleries []Gallery

	log.Printf("🔍 Запрос галерей для userID: %d - Получение метаданных и preview_url.", userID)

	query := `
		SELECT
			g.id,
			g.name,
			g.user_id,
			g.created_at,
			COALESCE(STRING_AGG(t.tag, ',' ORDER BY t.tag), '') AS tags_list,
			COALESCE(g.image_count, 0) AS image_count,
			g.preview_url
		FROM
			galleries g
		LEFT JOIN
			tags t ON g.id = t.gallery_id
		WHERE
			g.user_id = $1
		GROUP BY
			g.id, g.name, g.user_id, g.created_at, g.image_count, g.preview_url
		ORDER BY
			g.id DESC;
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("❌ Ошибка при запросе галерей для пользователя %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g Gallery
		var tagsStr sql.NullString

		if err := rows.Scan(
			&g.ID, &g.Name, &g.UserID, &g.CreatedAt, &tagsStr, &g.ImageCount, &g.PreviewURL,
		); err != nil {
			log.Printf("❌ Ошибка сканирования строки галереи для пользователя %d: %v", userID, err)
			continue
		}

		if tagsStr.Valid && tagsStr.String != "" {
			g.Tags = strings.Split(tagsStr.String, ",")
			for i, tag := range g.Tags {
				g.Tags[i] = strings.TrimSpace(tag)
			}
			g.Tags = filterEmptyStrings(g.Tags)
		} else {
			g.Tags = []string{}
		}

		if g.PreviewURL == "" {
			g.PreviewURL = "/static/no-image-placeholder.png"
		} else {
			if !strings.HasPrefix(g.PreviewURL, "/") {
				g.PreviewURL = "/" + g.PreviewURL
			}
		}
		galleries = append(galleries, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации по строкам галерей: %v", err)
		return nil, err
	}

	log.Printf("📊 Получено %d галерей для пользователя %d.", len(galleries), userID)
	return galleries, nil
}

func filterEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func GetMyGalleriesAPIHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ GetMyGalleriesAPIHandler: userID отсутствует в контексте после AuthMiddleware.")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован"})
			return
		}
		userID, ok := sessionUserID.(int64)
		if !ok || userID <= 0 {
			log.Printf("❌ GetMyGalleriesAPIHandler: Неверный формат или значение userID: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Неверный ID пользователя."})
			return
		}

		galleries, err := GetUserGalleries(db, userID)
		if err != nil {
			log.Printf("❌ GetMyGalleriesAPIHandler: Ошибка получения галерей для пользователя %d: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка загрузки ваших галерей."})
			return
		}

		log.Printf("✅ GetMyGalleriesAPIHandler: Отправлены %d галерей для пользователя %d.", len(galleries), userID)
		c.JSON(http.StatusOK, gin.H{
			"ok":        true,
			"galleries": galleries,
		})
	}
}

func DeleteGalleryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic в DeleteGalleryHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			}
		}()

		galleryName := c.Param("galleryName")
		if galleryName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Название галереи обязательно."})
			return
		}
		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ DeleteGalleryHandler: Пользователь не авторизован или userID не установлен")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован"})
			return
		}
		userID, ok := sessionUserID.(int64)
		if !ok || userID <= 0 {
			log.Printf("❌ DeleteGalleryHandler: Неверный формат или значение userID: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Неверный ID пользователя"})
			return
		}

		var folderPath string
		var galleryID int64
		err := db.QueryRow("SELECT id, folder_path FROM galleries WHERE name = $1 AND user_id = $2", galleryName, userID).Scan(&galleryID, &folderPath)
		if err == sql.ErrNoRows {
			log.Printf("⚠️ DeleteGalleryHandler: Галерея '%s' для пользователя %d не найдена.", galleryName, userID)
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "Галерея не найдена или у вас нет прав на ее удаление."})
			return
		}
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка запроса галереи '%s' для пользователя %d: %v", galleryName, userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при поиске галереи."})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка начала транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic в DeleteGalleryHandler во время транзакции: %v", r)
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			}
		}()

		_, err = tx.Exec("DELETE FROM favorites WHERE gallery_id = $1", galleryID)
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка удаления избранного для галереи ID=%d: %v", galleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка удаления из избранного."})
			return
		}

		_, err = tx.Exec("DELETE FROM gallery_images WHERE gallery_id = $1", galleryID)
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка удаления изображений для галереи ID=%d: %v", galleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка удаления изображений из базы данных."})
			return
		}

		_, err = tx.Exec("DELETE FROM tags WHERE gallery_id = $1", galleryID)
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка удаления тегов для галереи ID=%d: %v", galleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка удаления тегов."})
			return
		}

		_, err = tx.Exec("DELETE FROM galleries WHERE id = $1", galleryID)
		if err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка удаления галереи ID=%d из БД: %v", galleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка удаления галереи из базы данных."})
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка коммита транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при завершении операции."})
			return
		}

		log.Printf("DEBUG: Folder path from DB: '%s'", folderPath)
		log.Printf("DEBUG: UPLOADS_PHYSICAL_BASE_PATH: '%s'", UPLOADS_PHYSICAL_BASE_PATH)

		fullPathToDelete := filepath.Join(UPLOADS_PHYSICAL_BASE_PATH, folderPath)
		log.Printf("📂 DeleteGalleryHandler: Попытка удалить папку: %s", fullPathToDelete)

		if err := os.RemoveAll(fullPathToDelete); err != nil {
			log.Printf("❌ DeleteGalleryHandler: Ошибка удаления директории '%s': %v", fullPathToDelete, err)
		}

		log.Printf("✅ DeleteGalleryHandler: Галерея '%s' (ID: %d) успешно удалена для пользователя %d.", galleryName, galleryID, userID)
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Галерея успешно удалена!"})
	}
}
