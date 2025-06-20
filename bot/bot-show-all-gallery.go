package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func GetAllGalleries(db *sql.DB, viewerUserID int64, limit, offset int) ([]Gallery, error) {
	log.Printf("🔍 Запрос всех галерей (limit=%d, offset=%d) для пользователя %d - Использование БД для метаданных изображений", limit, offset, viewerUserID)

	var galleries []Gallery

	query := `
		SELECT
			g.id,
			g.name,
			g.user_id,
			g.created_at,
			COALESCE(g.image_count, 0) AS image_count,
			g.preview_url,
			u.telegram_user_id,
			u.telegram_username,
			u.first_name,
			u.last_name,
			u.photo_url,
			CASE WHEN f.user_id IS NOT NULL THEN TRUE ELSE FALSE END AS is_favorite
		FROM
			galleries g
		JOIN
			users u ON g.user_id = u.telegram_user_id
		LEFT JOIN
			favorites f ON g.id = f.gallery_id AND f.user_id = $3
		ORDER BY
			g.created_at DESC
		LIMIT $1 OFFSET $2;
	`
	log.Printf("🔍 SQL-запрос для GetAllGalleries: %s", query)

	rows, err := db.Query(query, limit, offset, viewerUserID)
	if err != nil {
		log.Printf("❌ Ошибка запроса галерей: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g Gallery
		var isFavorite bool
		if err := rows.Scan(
			&g.ID,
			&g.Name,
			&g.UserID,
			&g.CreatedAt,
			&g.ImageCount,
			&g.PreviewURL,
			&g.CreatorID,
			&g.CreatorUsername,
			&g.CreatorFirstName,
			&g.CreatorLastName,
			&g.CreatorPhotoURL,
			&isFavorite,
		); err != nil {
			log.Printf("❌ Ошибка чтения строки галереи: %v", err)
			continue
		}
		g.IsFavorite = isFavorite

		if g.PreviewURL == "" {
			g.PreviewURL = "/static/no-image-placeholder.png"
		}

		if !g.CreatorPhotoURL.Valid || g.CreatorPhotoURL.String == "" {
			g.CreatorPhotoURL = sql.NullString{String: "/static/default-avatar.png", Valid: true}
		}

		tagsQuery := "SELECT tag FROM tags WHERE gallery_id = $1"
		tagRows, err := db.Query(tagsQuery, g.ID)
		if err != nil {
			log.Printf("❌ Ошибка получения тегов для галереи ID=%d: %v", g.ID, err)
		} else {
			defer tagRows.Close()
			for tagRows.Next() {
				var tag string
				if err := tagRows.Scan(&tag); err != nil {
					log.Printf("❌ Ошибка сканирования тега: %v", err)
					continue
				}
				g.Tags = append(g.Tags, tag)
			}
		}
		galleries = append(galleries, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Ошибка при итерации строк галерей в GetAllGalleries: %v", err)
		return nil, err
	}
	log.Printf("📊 Получено %d галерей.", len(galleries))
	return galleries, nil
}

func GetGalleriesByTag(db *sql.DB, tagQuery string, viewerUserID int64, limit, offset int) ([]Gallery, error) {
	log.Printf("🔎 Поиск галерей по тегу: %s (limit=%d, offset=%d) для пользователя %d - Использование БД для метаданных изображений", tagQuery, limit, offset, viewerUserID)

	processedTagQuery := strings.ToLower(strings.TrimSpace(tagQuery))

	if processedTagQuery == "" || processedTagQuery == "null" {
		log.Println("▶ Запрос тега пуст или равен 'null', возвращаю все галереи.")
		return GetAllGalleries(db, viewerUserID, limit, offset)
	}

	query := `
		SELECT
			g.id,
			g.name,
			g.user_id,
			g.created_at,
			COALESCE(g.image_count, 0) AS image_count,
			g.preview_url,
			u.telegram_user_id,
			u.telegram_username,
			u.first_name,
			u.last_name,
			u.photo_url,
			CASE WHEN f.user_id IS NOT NULL THEN TRUE ELSE FALSE END AS is_favorite
		FROM
			galleries g
		JOIN tags t ON g.id = t.gallery_id
		JOIN users u ON g.user_id = u.telegram_user_id
		LEFT JOIN
			favorites f ON g.id = f.gallery_id AND f.user_id = $4
		WHERE LOWER(t.tag) LIKE '%' || $1 || '%'
		GROUP BY
			g.id, g.name, g.user_id, g.created_at, g.preview_url,
			u.telegram_user_id, u.telegram_username, u.first_name, u.last_name, u.photo_url, is_favorite
		ORDER BY
			g.created_at DESC
		LIMIT $2 OFFSET $3;
	`
	rows, err := db.Query(query, processedTagQuery, limit, offset, viewerUserID)
	if err != nil {
		log.Printf("❌ Ошибка выполнения запроса GetGalleriesByTag: %v", err)
		return nil, err
	}
	defer rows.Close()

	var galleries []Gallery

	for rows.Next() {
		var g Gallery
		var isFavorite bool
		if err := rows.Scan(
			&g.ID,
			&g.Name,
			&g.UserID,
			&g.CreatedAt,
			&g.ImageCount,
			&g.PreviewURL,
			&g.CreatorID,
			&g.CreatorUsername,
			&g.CreatorFirstName,
			&g.CreatorLastName,
			&g.CreatorPhotoURL,
			&isFavorite,
		); err != nil {
			log.Printf("❌ Ошибка чтения строки галереи в GetGalleriesByTag: %v", err)
			continue
		}
		g.IsFavorite = isFavorite

		if g.PreviewURL == "" {
			g.PreviewURL = "/static/no-image-placeholder.png"
		}
		if !g.CreatorPhotoURL.Valid || g.CreatorPhotoURL.String == "" {
			g.CreatorPhotoURL = sql.NullString{String: "/static/default-avatar.png", Valid: true}
		}

		tagsQuery := `SELECT tag FROM tags WHERE gallery_id = $1`
		tagRows, err := db.Query(tagsQuery, g.ID)
		if err != nil {
			log.Printf("❌ Ошибка получения тегов для галереи ID=%d в GetGalleriesByTag: %v", g.ID, err)
		} else {
			defer tagRows.Close()
			for tagRows.Next() {
				var tag string
				if err := tagRows.Scan(&tag); err != nil {
					log.Printf("❌ Ошибка сканирования тега: %v", err)
					continue
				}
				g.Tags = append(g.Tags, tag)
			}
		}
		galleries = append(galleries, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Ошибка при итерации строк галерей в GetGalleriesByTag: %v", err)
		return nil, err
	}
	log.Printf("📊 Найдено галерей по тегу '%s': %d", tagQuery, len(galleries))
	return galleries, nil
}

func GetGalleryImages(db *sql.DB, galleryID int64) ([]string, error) {
	log.Printf("🖼️ Запрос изображений для галереи ID: %d (из БД) - Получение full_size_image_path", galleryID)

	var images []string

	query := "SELECT full_size_image_path FROM gallery_images WHERE gallery_id = $1 ORDER BY id ASC"
	rows, err := db.Query(query, galleryID)
	if err != nil {
		log.Printf("❌ Ошибка запроса изображений из БД для галереи %d: %v", galleryID, err)
		return nil, fmt.Errorf("ошибка при получении URL изображений из БД: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var imageDBPath string
		if err := rows.Scan(&imageDBPath); err != nil {
			log.Printf("❌ Ошибка сканирования full_size_image_path для галереи %d: %v", galleryID, err)
			continue
		}
		images = append(images, imageDBPath)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Ошибка при итерации строк изображений: %v", err)
		return nil, err
	}
	log.Printf("✅ Загружено %d изображений для галереи ID: %d", len(images), galleryID)
	return images, nil
}
