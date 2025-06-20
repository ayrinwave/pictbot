package main

import (
	bot2 "Golang_Web_App_Bot/bot"
	db2 "Golang_Web_App_Bot/db"
	"Golang_Web_App_Bot/server"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"sync"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("❌ Ошибка при загрузке .env файла: %v", err)
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ Токен бота не найден в переменных окружения (TELEGRAM_BOT_TOKEN).")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("❌ Ошибка при создании Telegram-бота: %v", err)
	}
	log.Printf("✅ Бот авторизован как %s", bot.Self.UserName)

	db, err := db2.ConnectToDB()
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
	}
	// Отложенное закрытие соединения с базой данных
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("⚠️ Ошибка при закрытии соединения с базой данных: %v", cerr)
		}
	}()

	var wg sync.WaitGroup
	//счетчик ожидания
	wg.Add(2) // Одна горутина для бота, другая для веб-сервера счетчик ожидания

	// Запускаем веб-сервер в отдельной горутине
	go func() {
		defer wg.Done()
		server.StartServer(db, token) // Передаем db и token в StartServer
	}()

	// Запускаем обработчик обновлений Telegram-бота в отдельной горутине
	go func() {
		defer wg.Done()
		log.Println("🤖 Запуск обработчика обновлений Telegram...")
		bot2.HandleTelegramUpdates(bot, db)
	}()

	// Основная горутина ожидает завершения всех запущенных сервисов
	wg.Wait()
}
