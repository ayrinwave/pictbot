package bot

import (
	db2 "Golang_Web_App_Bot/db"
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
)

func HandleTelegramUpdates(bot *tgbotapi.BotAPI, dbConn *sql.DB) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				userID := update.Message.From.ID
				username := update.Message.From.UserName
				firstName := update.Message.From.FirstName
				lastName := update.Message.From.LastName
				photoURL := ""

				webAppBaseURL := os.Getenv("WEB_APP_URL")
				if webAppBaseURL == "" {
					log.Println("WEB_APP_URL не установлен")
					sendErrorMessage(bot, update.Message.Chat.ID, "Ошибка: веб-приложение недоступно.")
					continue
				}

				webAppURL := webAppBaseURL

				photoFilePath := "templates/static/image0.jpg"
				if err := sendWelcomePhoto(bot, update.Message.Chat.ID, photoFilePath); err != nil {
					log.Printf("Ошибка отправки фото: %v", err)
				}

				if err := sendWebAppMessage(bot, update.Message.Chat.ID, webAppURL); err != nil {
					log.Printf("Ошибка отправки сообщения с Web App кнопкой: %v", err)
				}

				dbUser, err := db2.FindOrCreateUser(
					dbConn,
					int64(userID),
					username,
					firstName,
					lastName,
					photoURL,
				)
				if err != nil {
					log.Printf("❌ Ошибка сохранения или получения пользователя в БД: %v", err)
					sendErrorMessage(bot, update.Message.Chat.ID, "Произошла ошибка при регистрации вас в системе.")
					continue
				}
				log.Printf("✅ Пользователь успешно обработан в БД: ID=%d, Username:%s, Name:%s %s",
					dbUser.TelegramUserID, dbUser.TelegramUsername.String, dbUser.FirstName.String, dbUser.LastName.String)

			default:
				sendErrorMessage(bot, update.Message.Chat.ID, "Неизвестная команда. Попробуйте /start.")
			}
		} else if update.CallbackQuery != nil {
		}
	}
}

func sendWelcomePhoto(bot *tgbotapi.BotAPI, chatID int64, photoPath string) error {
	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		return fmt.Errorf("файл %s не найден", photoPath)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(photoPath))
	photo.Caption = "Добро пожаловать в нашего Бота! В нем Вы сможете найти интересные картинки и опубликовать свои"

	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/start"),
		),
	)
	photo.ReplyMarkup = replyKeyboard

	_, err := bot.Send(photo)
	return err
}

func sendWebAppMessage(bot *tgbotapi.BotAPI, chatID int64, webAppURL string) error {
	inlineBtn := tgbotapi.InlineKeyboardButton{
		Text: "📱 Открыть Веб-приложение",
		WebApp: &tgbotapi.WebAppInfo{
			URL: webAppURL,
		},
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(inlineBtn),
	)

	msg := tgbotapi.NewMessage(chatID, "Вы можете запустить Веб-приложение по кнопке ниже:")
	msg.ReplyMarkup = inlineKeyboard

	_, err := bot.Send(msg)
	return err
}

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, _ = bot.Send(msg)
}
