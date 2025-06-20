package bot

import (
	db2 "Golang_Web_App_Bot/db"
	"bytes"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

const (
	MAX_CONCURRENT_IMAGE_PROCESSING = 1
	PREVIEW_WIDTH                   uint = 300
	FULL_SIZE_WIDTH                 uint = 1000
	JPEG_QUALITY_PREVIEW                 = 60
	JPEG_QUALITY_FULL                    = 80
)

var BotToken string

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("⚠️ Ошибка загрузки файла .env: %v (продолжаем, используя переменные окружения)", err)
	}

	BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if BotToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN не найден в переменных окружения или .env файле")
	}

	UPLOADS_BASE_PATH_FOR_WRITING = os.Getenv("UPLOAD_PATH")
	if UPLOADS_BASE_PATH_FOR_WRITING == "" {
		UPLOADS_BASE_PATH_FOR_WRITING = "/app/uploads"
		log.Printf("ℹ️ UPLOAD_PATH не установлен в переменных окружения. Используется путь по умолчанию: %s", UPLOADS_BASE_PATH_FOR_WRITING)
	} else {
		log.Printf("✅ UPLOAD_PATH из переменных окружения: %s", UPLOADS_BASE_PATH_FOR_WRITING)
	}

	if err := os.MkdirAll(UPLOADS_BASE_PATH_FOR_WRITING, 0755); err != nil {
		log.Fatalf("❌ Не удалось создать базовую директорию для загрузок '%s': %v", UPLOADS_BASE_PATH_FOR_WRITING, err)
	}
}

func saveProcessedImage(inputReader io.Reader, originalFilename string, fullGalleryFolderPath string, dbFolderPath string) (*ImagePaths, error) {
	imageData, err := io.ReadAll(inputReader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения исходного изображения '%s': %w", originalFilename, err)
	}

	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования изображения '%s': %w", originalFilename, err)
	}

	baseFileName := fmt.Sprintf("%s_%s",
		sanitizeFilename(strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename))),
		generateShortUUID())

	var resultPaths ImagePaths

	previewImg := resize.Resize(PREVIEW_WIDTH, 0, img, resize.Lanczos3)
	previewFileName := baseFileName + "_preview.jpg"
	previewAbsPath := filepath.Join(fullGalleryFolderPath, previewFileName)

	if err := saveJPG(previewImg, previewAbsPath, JPEG_QUALITY_PREVIEW); err != nil {
		return nil, fmt.Errorf("ошибка сохранения превью '%s': %w", previewAbsPath, err)
	}
	resultPaths.PreviewPath = filepath.ToSlash(filepath.Join(dbFolderPath, previewFileName))
	log.Printf("DEBUG: Сохранено превью: %s (DB: %s)", previewAbsPath, resultPaths.PreviewPath)

	var fullSizeFileName string
	var fullSizeAbsPath string

	switch format {
	case "png":
		fullSizeFileName = baseFileName + "_full.png"
		fullSizeAbsPath = filepath.Join(fullGalleryFolderPath, fullSizeFileName)
		fullSizeImg := resize.Resize(FULL_SIZE_WIDTH, 0, img, resize.Lanczos3)
		if err := savePNG(fullSizeImg, fullSizeAbsPath); err != nil {
			return nil, fmt.Errorf("ошибка сохранения полноразмерной версии PNG '%s': %w", fullSizeAbsPath, err)
		}
	case "gif":
		fullSizeFileName = baseFileName + "_full.gif"
		fullSizeAbsPath = filepath.Join(fullGalleryFolderPath, fullSizeFileName)

		gifImg, err := gif.DecodeAll(bytes.NewReader(imageData))
		if err != nil {
			return nil, fmt.Errorf("ошибка декодирования анимированного GIF '%s': %w", originalFilename, err)
		}

		if FULL_SIZE_WIDTH > 0 && uint(gifImg.Config.Width) > FULL_SIZE_WIDTH {
			newFrames := make([]*image.Paletted, len(gifImg.Image))
			for i, frame := range gifImg.Image {
				resizedFrame := resize.Resize(FULL_SIZE_WIDTH, 0, frame, resize.Lanczos3)
				if palettedFrame, ok := resizedFrame.(*image.Paletted); ok {
					newFrames[i] = palettedFrame
				} else {
					paletted := image.NewPaletted(resizedFrame.Bounds(), gifImg.Image[0].Palette)
					draw.Draw(paletted, paletted.Bounds(), resizedFrame, resizedFrame.Bounds().Min, draw.Src)
					newFrames[i] = paletted
				}
			}
			gifImg.Image = newFrames
			gifImg.Config.Width = int(FULL_SIZE_WIDTH)
			if len(newFrames) > 0 {
				gifImg.Config.Height = newFrames[0].Bounds().Dy()
			}
		}

		if err := saveAnimatedGIF(gifImg, fullSizeAbsPath); err != nil {
			return nil, fmt.Errorf("ошибка сохранения анимированной версии GIF '%s': %w", fullSizeAbsPath, err)
		}

	case "webp", "bmp":
		fullSizeFileName = baseFileName + "_full.jpg"
		fullSizeAbsPath = filepath.Join(fullGalleryFolderPath, fullSizeFileName)
		fullSizeImg := resize.Resize(FULL_SIZE_WIDTH, 0, img, resize.Lanczos3)
		if err := saveJPG(fullSizeImg, fullSizeAbsPath, JPEG_QUALITY_FULL); err != nil {
			return nil, fmt.Errorf("ошибка сохранения полноразмерной версии WebP/BMP в JPG '%s': %w", fullSizeAbsPath, err)
		}
	case "jpeg":
		fallthrough
	default:
		fullSizeFileName = baseFileName + "_full.jpg"
		fullSizeAbsPath = filepath.Join(fullGalleryFolderPath, fullSizeFileName)
		fullSizeImg := resize.Resize(FULL_SIZE_WIDTH, 0, img, resize.Lanczos3)
		if err := saveJPG(fullSizeImg, fullSizeAbsPath, JPEG_QUALITY_FULL); err != nil {
			return nil, fmt.Errorf("ошибка сохранения полноразмерной версии JPG '%s': %w", fullSizeAbsPath, err)
		}
	}

	resultPaths.FullSizePath = filepath.ToSlash(filepath.Join(dbFolderPath, fullSizeFileName))
	log.Printf("DEBUG: Сохранено полноразмерное изображение: %s (DB: %s)", fullSizeAbsPath, resultPaths.FullSizePath)

	return &resultPaths, nil
}

func saveAnimatedGIF(gifImg *gif.GIF, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	return gif.EncodeAll(out, gifImg)
}

func saveJPG(img image.Image, filePath string, quality int) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	var opt jpeg.Options
	opt.Quality = quality

	return jpeg.Encode(out, img, &opt)
}

func savePNG(img image.Image, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func AddGalleryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic в AddGalleryHandler: %v", r)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			}
		}()

		sessionUserID, exists := c.Get("userID")
		if !exists {
			log.Println("❌ AddGalleryHandler: Пользователь не авторизован или userID не установлен")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован"})
			return
		}
		userID, ok := sessionUserID.(int64)
		if !ok || userID <= 0 {
			log.Printf("❌ AddGalleryHandler: Неверный формат или значение userID: %v", sessionUserID)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Неверный ID пользователя"})
			return
		}

		telegramUsername, _ := c.Get("telegramUsername")
		usernameStr, _ := telegramUsername.(string)

		var existingUserDBID int64
		err := db.QueryRow("SELECT telegram_user_id FROM users WHERE telegram_user_id = $1", userID).Scan(&existingUserDBID)
		if err == sql.ErrNoRows {
			_, err := db.Exec("INSERT INTO users (telegram_user_id, telegram_username, created_at) VALUES ($1, $2, $3)",
				userID, usernameStr, time.Now())
			if err != nil {
				log.Printf("❌ AddGalleryHandler: Ошибка добавления нового пользователя ID=%d: %v", userID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка регистрации пользователя"})
				return
			}
			log.Printf("✅ AddGalleryHandler: Пользователь ID=%d добавлен в БД.", userID)
		} else if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка проверки существования пользователя ID=%d: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при проверке пользователя"})
			return
		}

		galleryName := c.PostForm("galleryName")
		if galleryName == "" {
			log.Println("❌ AddGalleryHandler: Название галереи не передано.")
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Название галереи обязательно"})
			return
		}
		cleanGalleryName := sanitizeFilename(galleryName)

		exists, err = GalleryExistsForUser(db, cleanGalleryName, userID)
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка проверки существования галереи '%s' для пользователя %d: %v", cleanGalleryName, userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при проверке галереи."})
			return
		}
		if exists {
			log.Printf("⚠️ AddGalleryHandler: Галерея '%s' уже существует для пользователя %d.", cleanGalleryName, userID)
			c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "Галерея с таким названием уже существует"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка начала транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				log.Printf("🚨 Panic в AddGalleryHandler во время транзакции: %v", r)
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Внутренняя ошибка сервера"})
			}
		}()
		var newGalleryID int64
		insertGalleryQuery := `INSERT INTO galleries (name, user_id, folder_path) VALUES ($1, $2, '') RETURNING id`
		err = tx.QueryRow(insertGalleryQuery, cleanGalleryName, userID).Scan(&newGalleryID)
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка сохранения галереи в БД (транзакция, первая вставка): %v", err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сохранения галереи в базу данных"})
			return
		}
		log.Printf("✅ AddGalleryHandler: Галерея '%s' (ID: %d) добавлена в БД (временный folder_path).", cleanGalleryName, newGalleryID)

		fullGalleryFolderPath := filepath.Join(UPLOADS_BASE_PATH_FOR_WRITING, "gallery_images", strconv.FormatInt(newGalleryID, 10))
		log.Printf("📂 AddGalleryHandler: Попытка создать папку: %s", fullGalleryFolderPath)

		if err := os.MkdirAll(fullGalleryFolderPath, 0755); err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка создания директории %s: %v", fullGalleryFolderPath, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при создании папки для галереи"})
			return
		}

		dbFolderPath := filepath.Join("gallery_images", strconv.FormatInt(newGalleryID, 10))
		log.Printf("DEBUG: folder_path, сохраняемый в БД: %s", dbFolderPath)

		updateFolderPathQuery := `UPDATE galleries SET folder_path = $1 WHERE id = $2`
		_, err = tx.Exec(updateFolderPathQuery, dbFolderPath, newGalleryID)
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка обновления folder_path для галереи ID=%d: %v", newGalleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка обновления пути к папке галереи"})
			return
		}
		log.Printf("✅ AddGalleryHandler: Обновлен folder_path: '%s' для галереи ID=%d.", dbFolderPath, newGalleryID)

		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка парсинга multipart формы: %v", err)
			tx.Rollback()
			if err := os.RemoveAll(fullGalleryFolderPath); err != nil {
				log.Printf("⚠️ AddGalleryHandler: Ошибка при удалении папки галереи '%s' после ошибки парсинга: %v", fullGalleryFolderPath, err)
			}
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Ошибка при обработке загруженных файлов"})
			return
		}

		files := form.File["galleryImages"]
		if len(files) == 0 {
			log.Println("❌ AddGalleryHandler: Нет загруженных файлов.")
			tx.Rollback()
			if err := os.RemoveAll(fullGalleryFolderPath); err != nil {
				log.Printf("⚠️ AddGalleryHandler: Ошибка при удалении пустой папки галереи '%s': %v", fullGalleryFolderPath, err)
			}
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Добавьте хотя бы одно изображение"})
			return
		}
		if len(files) > 10 {
			log.Printf("❌ AddGalleryHandler: Слишком много файлов (%d), максимум 10.", len(files))
			tx.Rollback()
			if err := os.RemoveAll(fullGalleryFolderPath); err != nil {
				log.Printf("⚠️ AddGalleryHandler: Ошибка при удалении папки галереи '%s': %v", fullGalleryFolderPath, err)
			}
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Вы можете загрузить не более 10 файлов."})
			return
		}

		const maxFileSize = 32 << 20
		var wg sync.WaitGroup
		var savedFileCount int32
		var imageErrors []error
		errChan := make(chan error, len(files))
		imagePathsToDBChan := make(chan *ImagePaths, len(files))

		sem := make(chan struct{}, MAX_CONCURRENT_IMAGE_PROCESSING)

		var firstGalleryPreviewURL atomic.Value
		firstGalleryPreviewURL.Store("")

		for _, fileHeader := range files {
			wg.Add(1)
			sem <- struct{}{}
			go func(fileHeader *multipart.FileHeader) {
				defer wg.Done()
				defer func() { <-sem }()

				if fileHeader.Size > maxFileSize {
					errChan <- fmt.Errorf("файл '%s' слишком большой (%s), пропущен", fileHeader.Filename, byteCountToHuman(fileHeader.Size))
					return
				}
				if !isImageFile(fileHeader.Filename) {
					errChan <- fmt.Errorf("файл '%s' не является изображением, пропущен", fileHeader.Filename)
					return
				}

				src, err := fileHeader.Open()
				if err != nil {
					errChan <- fmt.Errorf("ошибка открытия файла '%s': %v", fileHeader.Filename, err)
					return
				}
				defer src.Close()

				processedPaths, err := saveProcessedImage(src, fileHeader.Filename, fullGalleryFolderPath, dbFolderPath)
				if err != nil {
					errChan <- fmt.Errorf("ошибка обработки и сохранения изображения '%s': %v", fileHeader.Filename, err)
					return
				}

				currentSavedCount := atomic.AddInt32(&savedFileCount, 1)
				log.Printf("✅ AddGalleryHandler (Goroutine): Файл '%s' обработан и сохранен. Full: %s, Preview: %s",
					fileHeader.Filename, processedPaths.FullSizePath, processedPaths.PreviewPath)

				if currentSavedCount == 1 {
					firstGalleryPreviewURL.Store(processedPaths.PreviewPath)
					log.Printf("DEBUG: Первый файл для PreviewURL галереи: %s", processedPaths.PreviewPath)
				}

				imagePathsToDBChan <- processedPaths
			}(fileHeader)
		}

		wg.Wait()
		close(errChan)
		close(imagePathsToDBChan)

		for err := range errChan {
			imageErrors = append(imageErrors, err)
			log.Printf("❌ AddGalleryHandler: Ошибка обработки файла: %v", err)
		}

		if atomic.LoadInt32(&savedFileCount) == 0 {
			log.Printf("❌ AddGalleryHandler: Ни один файл не был успешно сохранен для галереи '%s'. Откат операции.", cleanGalleryName)
			if err := os.RemoveAll(fullGalleryFolderPath); err != nil {
				log.Printf("⚠️ AddGalleryHandler: Ошибка при удалении пустой папки галереи '%s': %v", fullGalleryFolderPath, err)
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не удалось сохранить ни одного изображения. Возможно, файлы неверного формата или размера."})
			return
		}

		finalPreviewURL := firstGalleryPreviewURL.Load().(string)

		if finalPreviewURL == "" {
			finalPreviewURL = "/static/no-image-placeholder.png"
		}

		updateGalleryQuery := `
			UPDATE galleries
			SET preview_url = $1, image_count = $2
			WHERE id = $3;
		`
		_, err = tx.Exec(updateGalleryQuery, finalPreviewURL, atomic.LoadInt32(&savedFileCount), newGalleryID)
		if err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка обновления preview_url/image_count для галереи ID=%d: %v", newGalleryID, err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка обновления данных галереи"})
			return
		}
		log.Printf("✅ AddGalleryHandler: Обновлен preview_url: '%s' и image_count: %d для галереи ID=%d.", finalPreviewURL, atomic.LoadInt32(&savedFileCount), newGalleryID)

		allProcessedPaths := make([]*ImagePaths, 0, atomic.LoadInt32(&savedFileCount))
		for processedPaths := range imagePathsToDBChan {
			allProcessedPaths = append(allProcessedPaths, processedPaths)
		}

		if len(allProcessedPaths) > 0 {
			stmt, err := tx.Prepare(`
				INSERT INTO gallery_images (gallery_id, full_size_image_path, preview_image_path, created_at)
				VALUES ($1, $2, $3, $4)
			`)
			if err != nil {
				log.Printf("❌ AddGalleryHandler: Ошибка подготовки запроса для массовой вставки изображений: %v", err)
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка подготовки вставки изображений"})
				return
			}
			defer stmt.Close()

			for _, p := range allProcessedPaths {
				_, err := stmt.Exec(newGalleryID, p.FullSizePath, p.PreviewPath, time.Now())
				if err != nil {
					log.Printf("❌ AddGalleryHandler: Ошибка сохранения путей изображений (full: '%s', preview: '%s') в БД для галереи %d: %v", p.FullSizePath, p.PreviewPath, newGalleryID, err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сохранения данных изображения в базу данных"})
					return
				}
			}
		}
		log.Printf("✅ AddGalleryHandler: Успешно сохранено %d путей изображений в БД для галереи ID=%d.", atomic.LoadInt32(&savedFileCount), newGalleryID)

		tagsInputStr := c.PostForm("tagsInput")
		if tagsInputStr != "" {
			tags := strings.Split(tagsInputStr, ",")
			cleanedTags := make([]string, 0, len(tags))
			for _, tag := range tags {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					cleanedTags = append(cleanedTags, trimmedTag)
				}
			}

			if len(cleanedTags) > 0 {
				err := AddTagsToGalleryTx(tx, newGalleryID, cleanedTags)
				if err != nil {
					log.Printf("❌ AddGalleryHandler: Ошибка добавления тегов для галереи ID=%d (транзакция): %v", newGalleryID, err)
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка добавления тегов к галерее"})
					return
				} else {
					log.Printf("✅ AddGalleryHandler: Добавлено %d тегов для галереи ID=%d.", len(cleanedTags), newGalleryID)
				}
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("❌ AddGalleryHandler: Ошибка коммита транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка сервера при завершении операции"})
			return
		}

		log.Printf("✅ AddGalleryHandler: Галерея '%s' (ID: %d) успешно создана для пользователя %d. Сохранено %d файлов и добавлены данные в БД.",
			cleanGalleryName, newGalleryID, userID, atomic.LoadInt32(&savedFileCount))

		if len(imageErrors) > 0 {
			c.JSON(http.StatusOK, gin.H{
				"ok":          false,
				"message":     fmt.Sprintf("Галерея '%s' создана, но некоторые файлы не были загружены успешно.", cleanGalleryName),
				"galleryName": cleanGalleryName,
				"imageCount":  atomic.LoadInt32(&savedFileCount),
				"galleryID":   newGalleryID,
				"userID":      userID,
				"errors":      imageErrors,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"ok":          true,
				"message":     "Галерея успешно создана!",
				"galleryName": cleanGalleryName,
				"imageCount":  atomic.LoadInt32(&savedFileCount),
				"galleryID":   newGalleryID,
				"userID":      userID,
			})
		}
	}
}

func byteCountToHuman(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
func generateShortUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}

var UPLOADS_BASE_PATH_FOR_WRITING string

func AddTagsToGalleryTx(tx *sql.Tx, galleryID int64, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	for _, tag := range tags {
		_, err := tx.Exec(
			"INSERT INTO tags (gallery_id, tag) VALUES ($1, $2)",
			galleryID,
			strings.ToLower(strings.TrimSpace(tag)),
		)
		if err != nil {
			return fmt.Errorf("ошибка добавления тега '%s': %w", tag, err)
		}
	}
	return nil
}

func GalleryExistsForUser(db *sql.DB, galleryName string, userID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM galleries WHERE LOWER(name) = LOWER($1) AND user_id = $2`
	err := db.QueryRow(query, galleryName, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки существования галереи: %w", err)
	}
	return count > 0, nil
}

func sanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[^\p{L}\p{N}\-_\.]+`)
	return reg.ReplaceAllString(filename, "_")
}

func AuthMiddleware(db *sql.DB, botToken string) gin.HandlerFunc {
	if botToken == "" {
		log.Fatal("❌ AuthMiddleware: TELEGRAM_BOT_TOKEN не был передан или является пустым.")
	}

	return func(c *gin.Context) {
		log.Printf("AuthMiddleware: Запрос: %s %s", c.Request.Method, c.Request.URL.Path)

		var initDataRaw string
		var parsedInitData initdata.InitData

		initDataRaw = c.GetHeader("X-Telegram-Init-Data")
		if initDataRaw != "" {
			log.Printf("DEBUG: AuthMiddleware: InitData получен из заголовка X-Telegram-Init-Data.")
		} else {
			if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
				var requestBody struct {
					InitData string `json:"initData"`
				}
				if err := c.ShouldBindJSON(&requestBody); err == nil {
					initDataRaw = requestBody.InitData
					if initDataRaw != "" {
						log.Printf("DEBUG: AuthMiddleware: InitData получен из JSON-тела.")
					} else {
						log.Printf("DEBUG: AuthMiddleware: JSON-тело обнаружено, но initData в нем пуст или не найден.")
					}
				} else {
					log.Printf("DEBUG: AuthMiddleware: Ошибка чтения JSON-тела для initData: %v", err)
				}
			}
		}

		if initDataRaw == "" && (c.Request.Method == "POST" || c.Request.Method == "PUT") {
			initDataRaw = c.PostForm("initData")
			if initDataRaw != "" {
				log.Printf("DEBUG: AuthMiddleware: InitData получен из PostForm.")
			} else {
				log.Println("DEBUG: AuthMiddleware: InitData не найден ни в JSON-теле, ни в PostForm.")
			}
		}

		if initDataRaw != "" {
			err := initdata.Validate(initDataRaw, botToken, 24*time.Hour)
			if err != nil {
				log.Printf("❌ AuthMiddleware: Ошибка валидации initData: %v", err)
				c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Неверные данные авторизации Telegram."})
				c.Abort()
				return
			}
			log.Println("✅ AuthMiddleware: initData успешно валидирован.")

			parsedInitData, err = initdata.Parse(initDataRaw)
			if err != nil {
				log.Printf("❌ AuthMiddleware: Ошибка парсинга initData после валидации: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка обработки данных пользователя."})
				c.Abort()
				return
			}

			if parsedInitData.User.ID == 0 {
				log.Println("❌ AuthMiddleware: В валидном initData отсутствует информация о пользователе (User - nil или User.ID=0).")
				c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "user data missing or invalid in initData"})
				c.Abort()
				return
			}

			userID := parsedInitData.User.ID
			username := parsedInitData.User.Username
			firstName := parsedInitData.User.FirstName
			lastName := parsedInitData.User.LastName
			photoURL := parsedInitData.User.PhotoURL

			log.Printf("✅ AuthMiddleware: initData успешно обработан для пользователя %d. UserID: %d, Username: %s, Name: %s %s, PhotoURL: %s",
				userID, userID, username, firstName, lastName, photoURL)

			dbUser, err := db2.FindOrCreateUser(db, userID, username, firstName, lastName, photoURL)
			if err != nil {
				log.Printf("❌ AuthMiddleware: Ошибка при FindOrCreateUser: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Ошибка при регистрации пользователя в системе."})
				c.Abort()
				return
			}
			log.Printf("✅ AuthMiddleware: Пользователь БД (ID: %d) успешно обновлен/создан.", dbUser.TelegramUserID)

			c.Set("userID", userID)
			c.Set("telegramUsername", dbUser.TelegramUsername.String)
			c.Set("firstName", dbUser.FirstName.String)
			c.Set("lastName", dbUser.LastName.String)
			c.Set("photoURL", dbUser.PhotoURL.String)
			c.Set("dbUser", dbUser)

		} else {
			log.Println("❌ AuthMiddleware: InitData отсутствует в запросе. Требуется авторизация.")
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "Пользователь не авторизован"})
			c.Abort()
			return
		}

		c.Next()
	}
}
