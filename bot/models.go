// bot/models.go
package bot

import (
	"database/sql"
	"time"
)

// Gallery - Каноническая структура для представления данных галереи.
// Используется как для внутреннего представления, так и для API-ответа.
// Поле 'Images' здесь отсутствует, так как изображения загружаются лениво.
// 'FolderPath' также убран из JSON-ответа, так как это внутренний путь на сервере.
type Gallery struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	UserID     int64     `json:"userID"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"createdAt"`
	ImageCount int       `json:"imageCount"`
	PreviewURL string    `json:"previewURL"` // URL для обложки/первого фото

	// --- НОВЫЕ ПОЛЯ ДЛЯ ИНФОРМАЦИИ О СОЗДАТЕЛЕ ---
	CreatorID        int64          `json:"creatorID"`        // ID создателя (дублирует UserID, но для ясности)
	CreatorUsername  sql.NullString `json:"creatorUsername"`  // Имя пользователя Telegram
	CreatorFirstName sql.NullString `json:"creatorFirstName"` // Имя пользователя (first_name)
	CreatorLastName  sql.NullString `json:"creatorLastName"`  // Фамилия пользователя (last_name)
	CreatorPhotoURL  sql.NullString `json:"creatorPhotoURL"`  // URL аватара пользователя
	IsFavorite       bool           `json:"isFavorite"`
}

// UserProfile представляет профиль пользователя Telegram.
type UserProfile struct {
	ID          int64          `json:"id"`
	Username    sql.NullString `json:"username"`
	FirstName   sql.NullString `json:"first_name"`
	LastName    sql.NullString `json:"last_name"`
	PhotoURL    sql.NullString `json:"photo_url"`
	CreatedAt   time.Time      `json:"created_at"`
	LastLoginAt time.Time      `json:"last_login_at"`
}

//type DBUser struct {
//	TelegramUserID   int64
//	TelegramUsername sql.NullString
//	FirstName        sql.NullString
//	LastName         sql.NullString
//	PhotoURL         sql.NullString
//	CreatedAt        time.Time
//}

// InitDataUser и InitData остаются без изменений
type InitDataUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type InitData struct {
	QueryID  string       `json:"query_id"`
	User     InitDataUser `json:"user"`
	AuthDate int64        `json:"auth_date"`
	Hash     string       `json:"hash"`
}
type ImagePaths struct {
	FullSizePath string
	PreviewPath  string
}

//	type Subscription struct {
//		SubscriberID int64     `json:"subscriber_id"`  // ID пользователя, который подписывается
//		TargetUserID int64     `json:"target_user_id"` // ID пользователя, на которого подписываются
//		CreatedAt    time.Time `json:"created_at"`
//	}
type SubscribedUserProfile struct {
	TelegramUserID   int64          `json:"telegram_user_id"`
	TelegramUsername sql.NullString `json:"telegram_username"`
	FirstName        sql.NullString `json:"first_name"`
	LastName         sql.NullString `json:"last_name"`
	PhotoURL         sql.NullString `json:"photo_url"`
}
type GalleryFullDetail struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	UserID     int64     `json:"user_id"` // ID создателя галереи
	PreviewURL string    `json:"previewURL"`
	ImageCount int       `json:"imageCount"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"createdAt"`
	// Поля для информации о создателе галереи
	CreatorID        int64          `json:"creatorID"` // Telegram User ID создателя
	CreatorUsername  sql.NullString `json:"creatorUsername"`
	CreatorFirstName sql.NullString `json:"creatorFirstName"`
	CreatorLastName  sql.NullString `json:"creatorLastName"`
	CreatorPhotoURL  sql.NullString `json:"creatorPhotoURL"`
}
