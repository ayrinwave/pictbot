package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// GetUserProfileByID остается без изменений, если UserProfile определена.
// Убедитесь, что она возвращает *UserProfile.
// bot/gallery_queries.go
// bot/gallery_queries.go (или где у вас находится GetUserProfileByID)
func GetUserProfileByID(db *sql.DB, userID int64) (*UserProfile, error) {
	var user UserProfile
	query := `
        SELECT
            telegram_user_id, -- Используйте реальное имя столбца ID в вашей таблице users
            telegram_username,
            first_name,
            last_name,
            photo_url
        FROM
            users -- Используйте имя вашей таблицы пользователей
        WHERE
            telegram_user_id = $1` // <-- ИЗМЕНЕНО: ИСПОЛЬЗУЕМ $1

	row := db.QueryRow(query, userID)
	err := row.Scan(
		&user.ID, // Map to telegram_user_id
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.PhotoURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		return nil, fmt.Errorf("error scanning user profile: %w", err)
	}
	return &user, nil
}

// GetGalleriesByUserID - модифицирована для использования viewerUserID для статуса избранного
// и для возврата галерей конкретного пользователя.
func GetGalleriesByUserID(db *sql.DB, targetUserID int64, searchQuery string, viewerUserID int64) ([]Gallery, error) {
	log.Printf("🔎 Поиск галерей пользователя ID: %d по тегу: %s (для пользователя %d) - Использование БД для метаданных изображений", targetUserID, searchQuery, viewerUserID)

	var galleries []Gallery
	var queryBuilder strings.Builder
	args := []interface{}{}
	argCounter := 1 // Начинаем с 1 для позиционных параметров SQL

	// Основная часть SELECT и FROM
	queryBuilder.WriteString(`
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
            favorites f ON g.id = f.gallery_id AND f.user_id = $` + strconv.Itoa(argCounter) + `
    `)
	args = append(args, viewerUserID)
	argCounter++

	// Условное добавление JOIN для тегов (ДО WHERE)
	if searchQuery != "" {
		queryBuilder.WriteString(`
        JOIN tags t ON g.id = t.gallery_id
        `)
	}

	// НАЧАЛО УСЛОВИЙ WHERE
	queryBuilder.WriteString(` WHERE `)
	queryBuilder.WriteString(`g.user_id = $` + strconv.Itoa(argCounter))
	args = append(args, targetUserID)
	argCounter++

	// Добавление условия для тега, если searchQuery не пуст (ПОСЛЕ WHERE)
	if searchQuery != "" {
		queryBuilder.WriteString(`
        AND LOWER(t.tag) LIKE '%' || $` + strconv.Itoa(argCounter) + ` || '%'
        `)
		args = append(args, strings.ToLower(strings.TrimSpace(searchQuery)))
		argCounter++
	}

	// GROUP BY и ORDER BY
	queryBuilder.WriteString(`
        GROUP BY
            g.id, g.name, g.user_id, g.created_at, g.preview_url,
            u.telegram_user_id, u.telegram_username, u.first_name, u.last_name, u.photo_url, is_favorite
        ORDER BY
            g.created_at DESC;
    `)

	finalQuery := queryBuilder.String()
	log.Printf("🔍 SQL-запрос для GetGalleriesByUserID: %s", finalQuery)
	log.Printf("🔍 Аргументы SQL: %+v", args)

	rows, err := db.Query(finalQuery, args...)
	if err != nil {
		log.Printf("❌ Ошибка выполнения запроса GetGalleriesByUserID: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g Gallery
		var isFavorite bool // Для сканирования значения is_favorite
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
			&isFavorite, // <-- Сканируем is_favorite
		); err != nil {
			log.Printf("❌ Ошибка чтения строки галереи в GetGalleriesByUserID: %v", err)
			continue
		}
		g.IsFavorite = isFavorite // Присваиваем сканированное значение

		if g.PreviewURL == "" {
			g.PreviewURL = "/static/no-image-placeholder.png"
		}
		// Убедитесь, что CreatorPhotoURL обрабатывается правильно, если он NullString
		if !g.CreatorPhotoURL.Valid || g.CreatorPhotoURL.String == "" {
			g.CreatorPhotoURL = sql.NullString{String: "/static/default_avatar.png", Valid: true}
		}

		// Получаем теги для текущей галереи
		tagsQuery := `SELECT tag FROM tags WHERE gallery_id = $1`
		tagRows, err := db.Query(tagsQuery, g.ID)
		if err != nil {
			log.Printf("❌ Ошибка получения тегов для галереи ID=%d в GetGalleriesByUserID: %v", g.ID, err)
		} else {
			// Обязательно закрывайте tagRows
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
		log.Printf("❌ Ошибка при итерации строк галерей в GetGalleriesByUserID: %v", err)
		return nil, err
	}
	log.Printf("📊 Найдено %d галерей для пользователя %d.", len(galleries), targetUserID)
	return galleries, nil
}
