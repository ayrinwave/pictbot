package db

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"os"
	"time"
)

type Gallery struct {
	ID          int64
	UserID      int64
	GalleryName string
	Photos      []string
	CreatedAt   time.Time
	Name        string
}

type DBUser struct {
	TelegramUserID   int64
	TelegramUsername sql.NullString
	FirstName        sql.NullString
	LastName         sql.NullString
	PhotoURL         sql.NullString
	CreatedAt        time.Time
}

func ConnectToDB() (*sql.DB, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Ошибка загрузки файла .env: ", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка при открытии подключения к базе данных: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}

	log.Println("✅ Подключение к базе данных установлено.")
	return db, nil
}

func FindOrCreateUser(db *sql.DB,
	telegramUserID int64,
	username string,
	firstName string,
	lastName string,
	photoURL string,
) (*DBUser, error) {
	var user DBUser

	err := db.QueryRow(`
		SELECT telegram_user_id, telegram_username, first_name, last_name, photo_url, created_at
		FROM users
		WHERE telegram_user_id = $1`, telegramUserID).
		Scan(&user.TelegramUserID, &user.TelegramUsername, &user.FirstName, &user.LastName, &user.PhotoURL, &user.CreatedAt)

	if err == sql.ErrNoRows {
		log.Printf("DEBUG: Пользователь ID=%d не найден, создаем новую запись.", telegramUserID)
		now := time.Now()
		_, err := db.Exec(`
			INSERT INTO users (telegram_user_id, telegram_username, first_name, last_name, photo_url, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			telegramUserID,
			sql.NullString{String: username, Valid: username != ""},
			sql.NullString{String: firstName, Valid: firstName != ""},
			sql.NullString{String: lastName, Valid: lastName != ""},
			sql.NullString{String: photoURL, Valid: photoURL != ""},
			now)
		if err != nil {
			return nil, fmt.Errorf("ошибка при добавлении пользователя в БД: %v", err)
		}

		user.TelegramUserID = telegramUserID
		user.TelegramUsername = sql.NullString{String: username, Valid: username != ""}
		user.FirstName = sql.NullString{String: firstName, Valid: firstName != ""}
		user.LastName = sql.NullString{String: lastName, Valid: lastName != ""}
		user.PhotoURL = sql.NullString{String: photoURL, Valid: photoURL != ""}
		user.CreatedAt = now

		log.Printf("✅ Пользователь ID=%d добавлен в БД. Username: %s, Name: %s %s",
			user.TelegramUserID, user.TelegramUsername.String, user.FirstName.String, user.LastName.String)
	} else if err != nil {
		return nil, fmt.Errorf("ошибка при проверке/поиске пользователя ID=%d в БД: %v", telegramUserID, err)
	} else {
		log.Printf("DEBUG: Пользователь ID=%d найден, обновляем его данные.", telegramUserID)
		_, err := db.Exec(`
			UPDATE users
			SET telegram_username = $1, first_name = $2, last_name = $3, photo_url = $4
			WHERE telegram_user_id = $5`,
			sql.NullString{String: username, Valid: username != ""},
			sql.NullString{String: firstName, Valid: firstName != ""},
			sql.NullString{String: lastName, Valid: lastName != ""},
			sql.NullString{String: photoURL, Valid: photoURL != ""},
			telegramUserID)
		if err != nil {
			return nil, fmt.Errorf("ошибка при обновлении данных пользователя ID=%d: %v", telegramUserID, err)
		}
		user.TelegramUsername = sql.NullString{String: username, Valid: username != ""}
		user.FirstName = sql.NullString{String: firstName, Valid: firstName != ""}
		user.LastName = sql.NullString{String: lastName, Valid: lastName != ""}
		user.PhotoURL = sql.NullString{String: photoURL, Valid: photoURL != ""}
		log.Printf("✅ Данные пользователя ID=%d обновлены в БД. Username: %s, Name: %s %s",
			user.TelegramUserID, user.TelegramUsername.String, user.FirstName.String, user.LastName.String)
	}

	return &user, nil
}
