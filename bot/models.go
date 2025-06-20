package bot

import (
	"database/sql"
	"time"
)

type Gallery struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	UserID          int64          `json:"userID"`
	Tags            []string       `json:"tags"`
	CreatedAt       time.Time      `json:"createdAt"`
	ImageCount      int            `json:"imageCount"`
	PreviewURL      string         `json:"previewURL"`
	CreatorID       int64          `json:"creatorID"`
	CreatorUsername sql.NullString `json:"creatorUsername"`
	CreatorFirstName sql.NullString `json:"creatorFirstName"`
	CreatorLastName  sql.NullString `json:"creatorLastName"`
	CreatorPhotoURL  sql.NullString `json:"creatorPhotoURL"`
	IsFavorite      bool           `json:"isFavorite"`
}

type UserProfile struct {
	ID          int64          `json:"id"`
	Username    sql.NullString `json:"username"`
	FirstName   sql.NullString `json:"first_name"`
	LastName    sql.NullString `json:"last_name"`
	PhotoURL    sql.NullString `json:"photo_url"`
	CreatedAt   time.Time      `json:"created_at"`
	LastLoginAt time.Time      `json:"last_login_at"`
}

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

type SubscribedUserProfile struct {
	TelegramUserID   int64          `json:"telegram_user_id"`
	TelegramUsername sql.NullString `json:"telegram_username"`
	FirstName        sql.NullString `json:"first_name"`
	LastName         sql.NullString `json:"last_name"`
	PhotoURL         sql.NullString `json:"photo_url"`
}
type GalleryFullDetail struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	UserID          int64          `json:"user_id"`
	PreviewURL      string         `json:"previewURL"`
	ImageCount      int            `json:"imageCount"`
	Tags            []string       `json:"tags"`
	CreatedAt       time.Time      `json:"createdAt"`
	CreatorID       int64          `json:"creatorID"`
	CreatorUsername sql.NullString `json:"creatorUsername"`
	CreatorFirstName sql.NullString `json:"creatorFirstName"`
	CreatorLastName  sql.NullString `json:"creatorLastName"`
	CreatorPhotoURL  sql.NullString `json:"creatorPhotoURL"`
}
