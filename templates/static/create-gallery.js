// static/create-gallery.js

let selectedFiles = [];

const galleryForm = document.getElementById('galleryForm');
const galleryImagesInput = document.getElementById('galleryImages');
const imagePreviewsContainer = document.getElementById('imagePreviews');
const errorMessageDiv = document.getElementById('errorMessage');
const submitButton = galleryForm.querySelector('button[type="submit"]');

const customModal = document.getElementById('customModal');
const modalMessage = document.getElementById('modalMessage');
const closeModalButton = document.getElementById('closeModal');
const okModalButton = document.getElementById('okModalBtn');

const imageViewerModal = document.getElementById('myModal');
const modalImage = document.getElementById('modalImage');
const closeImageViewerButton = document.querySelector('.close-image-modal');

const tg = window.Telegram.WebApp;
tg.ready();

// --- Глобальный массив для накопления логов ---
let debugLogs = [];

const originalConsoleLog = console.log;
const originalConsoleError = console.error;
const originalConsoleWarn = console.warn;

console.log = function(...args) {
    const message = args.map(arg => {
        if (typeof arg === 'object' && arg !== null) {
            try {
                // Пытаемся получить больше информации об объекте File, если это он
                if (arg instanceof File || (arg && typeof arg.name === 'string' && typeof arg.size === 'number')) {
                    return `File {name: "${arg.name}", size: ${arg.size}, type: "${arg.type}", lastModified: ${arg.lastModified}}`;
                }
                return JSON.stringify(arg, null, 2);
            } catch (e) {
                return `[Object: ${arg.constructor.name}]`;
            }
        }
        return String(arg);
    }).join(' ');
    debugLogs.push(`[LOG]: ${message}`);
    if (debugLogs.length > 50) debugLogs.shift();
    originalConsoleLog.apply(console, args);
};

console.error = function(...args) {
    const message = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)).join(' ');
    debugLogs.push(`[ERROR]: ${message}`);
    if (debugLogs.length > 50) debugLogs.shift();
    originalConsoleError.apply(console, args);
};

console.warn = function(...args) {
    const message = args.map(arg => typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)).join(' ');
    debugLogs.push(`[WARN]: ${message}`);
    if (debugLogs.length > 50) debugLogs.shift();
    originalConsoleWarn.apply(console, args);
};

function showCustomModal(message, callback, includeLogs = false) {
    if (!customModal || !modalMessage || !closeModalButton || !okModalButton) {
        alert("Ошибка модального окна: " + message);
        if (callback) callback();
        return;
    }

    let fullMessage = message;
    if (includeLogs && debugLogs.length > 0) {
        fullMessage += "\n\n--- Debug Logs ---\n" + debugLogs.join('\n');
    }
    modalMessage.textContent = fullMessage;
    customModal.style.display = 'flex';

    closeModalButton.onclick = null;
    okModalButton.onclick = null;
    customModal.onclick = null;

    const handler = () => {
        closeCustomModal();
        if (callback && typeof callback === 'function') {
            callback();
        }
    };

    closeModalButton.onclick = handler;
    okModalButton.onclick = handler;
    customModal.onclick = (event) => {
        if (event.target === customModal) {
            handler();
        }
    };
}

function closeCustomModal() {
    if (customModal) {
        customModal.style.display = 'none';
    }
}

function openImageModal(src) {
    if (!imageViewerModal || !modalImage) {
        console.error('Modal elements for image viewer not found. Check HTML IDs.');
        return;
    }
    modalImage.src = src;
    imageViewerModal.style.display = 'flex';
}

function closeImageModal() {
    if (imageViewerModal) {
        imageViewerModal.style.display = 'none';
    }
}

if (closeImageViewerButton) {
    closeImageViewerButton.addEventListener('click', closeImageModal);
}
if (imageViewerModal) {
    imageViewerModal.addEventListener('click', (e) => {
        if (e.target === imageViewerModal) {
            closeImageModal();
        }
    });
}

function clearErrorMessage() {
    if (errorMessageDiv) {
        errorMessageDiv.textContent = '';
    }
}

function renderImagePreviews() {
    if (!imagePreviewsContainer) {
        console.error('Контейнер для предпросмотров изображений не найден. Проверьте HTML ID.');
        return;
    }
    imagePreviewsContainer.innerHTML = '';

    console.log('renderImagePreviews: selectedFiles.length =', selectedFiles.length);

    if (selectedFiles.length === 0) {
        imagePreviewsContainer.style.display = 'none';
        return;
    } else {
        imagePreviewsContainer.style.display = 'grid';
    }

    selectedFiles.forEach((file) => {
        // Дополнительная проверка на валидность объекта File перед чтением
        if (!(file instanceof File) || !file.name || file.size === undefined || !file.type) {
            console.error('renderImagePreviews: Обнаружен невалидный объект File:', file);
            showCustomModal(`Не могу отобразить предпросмотр файла. Возможно, файл недоступен или поврежден: ${file.name || 'Без имени'}`, null, true);
            return; // Пропускаем этот файл
        }

        const reader = new FileReader();
        reader.onload = (e) => {
            console.log('FileReader loaded for file:', file.name, 'Size:', file.size);

            const previewItem = document.createElement('div');
            previewItem.classList.add('preview-item');

            const img = document.createElement('img');
            img.src = e.target.result;
            img.alt = `Предпросмотр ${file.name}`;
            img.classList.add('preview-image');

            const removeButton = document.createElement('button');
            removeButton.classList.add('remove-image-button');
            removeButton.innerHTML = '&times;';
            removeButton.title = 'Открепить фото';
            removeButton.addEventListener('click', (e) => {
                e.preventDefault();

                const currentIndex = selectedFiles.findIndex(f => f === file);
                if (currentIndex > -1) {
                    selectedFiles.splice(currentIndex, 1);
                }
                console.log('Файл удален. Новое количество:', selectedFiles.length);
                renderImagePreviews();
            });

            const viewButton = document.createElement('button');
            viewButton.classList.add('view-image-button');
            viewButton.innerHTML = '👁️';
            viewButton.title = 'Просмотреть фото';
            viewButton.addEventListener('click', (e) => {
                e.preventDefault();
                openImageModal(img.src);
            });

            previewItem.appendChild(img);
            previewItem.appendChild(removeButton);
            previewItem.appendChild(viewButton);
            imagePreviewsContainer.appendChild(previewItem);
            console.log('Элементы предпросмотра добавлены для:', file.name);
        };
        reader.onerror = (e) => {
            console.error('Ошибка чтения файла:', file.name, e);
            showCustomModal(`Не удалось прочитать файл ${file.name}. Ошибка: ${e.message || 'Неизвестная ошибка'}.`, null, true);
        };
        reader.readAsDataURL(file);
    });
}

if (galleryImagesInput) {
    galleryImagesInput.addEventListener('change', (event) => {
        console.log('Событие change сработало на galleryImagesInput!');
        clearErrorMessage();

        const newFiles = Array.from(event.target.files);
        console.log('Получен FileList с длиной:', newFiles.length);
        console.log('Содержимое newFiles:', newFiles.map(f => {
            if (f instanceof File) {
                return { name: f.name, size: f.size, type: f.type, lastModified: f.lastModified };
            }
            return f; // Вернет пустой объект, если File невалиден
        }));

        if (newFiles.length === 0) {
            console.log('Файлы не выбраны или FileList пуст.');
            galleryImagesInput.value = "";
            showCustomModal("Выбор файлов отменен или не удалось получить файлы. Пожалуйста, попробуйте снова.", null, true);
            return;
        }

        let filesToAdd = [];
        let rejectedCount = 0;

        for (const file of newFiles) {
            // КЛЮЧЕВАЯ ПРОВЕРКА: Если объект File невалиден, пропускаем его
            if (!(file instanceof File) || !file.name || file.size === undefined || !file.type) {
                console.error(`Обнаружен невалидный объект File (Галерея?): ${JSON.stringify(file)}`);
                showCustomModal(`Ошибка: Выбранный файл (${file.name || 'без имени'}) не может быть обработан. Возможно, из-за особенностей выбора из галереи.`, null, true);
                rejectedCount++;
                continue;
            }

            // Проверка на дубликаты
            const isDuplicate = selectedFiles.some(
                (existingFile) => existingFile.name === file.name && existingFile.size === file.size
            );

            if (isDuplicate) {
                console.warn(`Файл ${file.name} уже выбран и будет проигнорирован.`);
                rejectedCount++;
                continue;
            }

            // Проверка на максимальное количество файлов
            if (selectedFiles.length + filesToAdd.length >= 10) {
                showCustomModal("Вы можете загрузить не более 10 файлов! Удалите одно из выбранных, чтобы добавить новое.", null, true);
                rejectedCount++;
                break;
            }

            // Проверка типа файла
            if (!file.type.startsWith('image/')) {
                showCustomModal(`Файл ${file.name} не является изображением и будет проигнорирован.`, null, true);
                rejectedCount++;
                continue;
            }
            // Проверка размера файла (32 MB)
            if (file.size > 32 * 1024 * 1024) {
                showCustomModal(`Файл ${file.name} превышает 32MB!`, null, true);
                rejectedCount++;
                continue;
            }

            filesToAdd.push(file);
            console.log('Файл готов к добавлению:', file.name);
        }

        selectedFiles = selectedFiles.concat(filesToAdd);
        console.log('Итоговый массив selectedFiles:', selectedFiles.length, selectedFiles);

        renderImagePreviews();

        galleryImagesInput.value = "";

        if (rejectedCount > 0) {
            showCustomModal(`Некоторые файлы были проигнорированы. Успешно добавлено: ${filesToAdd.length}`, null, true);
        }
    });
}

// Запускаем логику после загрузки DOM
document.addEventListener('DOMContentLoaded', function() {
    if (!galleryForm || !submitButton || !errorMessageDiv) {
        console.error('Основные элементы формы не найдены. Проверьте HTML ID.');
        showCustomModal("Ошибка: Не найдены основные элементы формы. Приложение не может быть загружено корректно.", null, true);
        return;
    }

    if (tg) {
        tg.ready();
        console.log("Telegram WebApp is initialized.");
        console.log("initData:", tg.initData ? JSON.stringify(tg.initData).substring(0, 100) + '...' : "initData is null/undefined");

        fetch('/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ initData: tg.initData })
        })
            .then(res => {
                if (!res.ok) {
                    return res.json().then(errData => Promise.reject(errData.error || `HTTP error! status: ${res.status}`));
                }
                return res.json();
            })
            .then(data => {
                if (data.ok) {
                    const telegramUserIDInput = document.getElementById('telegramUserID');
                    const telegramUsernameInput = document.getElementById('telegramUsername');

                    if (telegramUserIDInput) telegramUserIDInput.value = data.user.id;
                    if (telegramUsernameInput) telegramUsernameInput.value = data.user.username || '';

                    console.log('✅ Telegram user authenticated:', data.user.id); // Упрощено для лога
                    submitButton.disabled = false;
                } else {
                    showCustomModal("Ошибка авторизации через Telegram: " + (data.error || 'Неизвестная ошибка'), null, true);
                    submitButton.disabled = true;
                    console.error('❌ Ошибка авторизации:', data.error);
                }
            })
            .catch(error => {
                showCustomModal("Ошибка запроса к /auth: " + (error.message || error), null, true);
                submitButton.disabled = true;
                console.error('❌ Ошибка запроса к /auth:', error);
            });
    } else {
        console.warn("Telegram WebApp is not available. Running in a standard browser environment.");
        showCustomModal("Это приложение предназначено для запуска в Telegram. Авторизация недоступна.", null, true);
        submitButton.disabled = true;
    }
});

// Обработчик отправки формы
if (galleryForm) {
    galleryForm.addEventListener('submit', async function (event) {
        event.preventDefault();

        clearErrorMessage();

        if (submitButton.disabled) {
            showCustomModal("Форма заблокирована из-за проблем с авторизацией.", null, true);
            return;
        }

        const telegramUserID = document.getElementById('telegramUserID').value;
        if (!telegramUserID || telegramUserID === 'null' || telegramUserID === '') {
            showCustomModal("Ошибка: Telegram ID не получен, невозможно создать галерею. Пожалуйста, попробуйте перезапустить приложение в Telegram.", null, true);
            return;
        }

        const galleryName = document.getElementById('galleryName').value.trim();
        if (!galleryName) {
            showCustomModal("Пожалуйста, введите название галереи.", null, true);
            return;
        }

        // Проверяем выбранные файлы на валидность перед отправкой
        const validFiles = selectedFiles.filter(file => file instanceof File && file.name && file.size !== undefined && file.type);
        if (validFiles.length === 0) {
            showCustomModal("Пожалуйста, выберите хотя бы одно корректное изображение.", null, true);
            return;
        }


        const tagsInput = document.getElementById('tagsInput').value.trim();
        const tagsArray = tagsInput.split(',').map(tag => tag.trim()).filter(tag => tag !== '');

        const formData = new FormData();
        formData.append('galleryName', galleryName);

        tagsArray.forEach(tag => {
            formData.append("tagsInput", tag);
        });

        validFiles.forEach((file) => { // Отправляем только валидные файлы
            formData.append('galleryImages', file);
        });

        if (window.Telegram && window.Telegram.WebApp && window.Telegram.WebApp.initData) {
            formData.append('initData', window.Telegram.WebApp.initData);
        } else {
            console.error("Telegram WebApp initData не доступен при отправке формы!");
            showCustomModal("Ошибка: Нет данных авторизации Telegram. Пожалуйста, перезапустите приложение.", null, true);
            return;
        }

        try {
            const response = await fetch('/api/add_gallery', {
                method: 'POST',
                body: formData,
            });

            const contentType = response.headers.get("content-type");
            let result;

            if (contentType && contentType.includes("application/json")) {
                result = await response.json();
            } else {
                result = await response.text();
                throw new Error(`Сервер вернул не JSON ответ: ${result}`);
            }

            if (response.ok && result.ok) {
                showCustomModal(`Галерея "${result.galleryName}" успешно создана с ${result.imageCount} файлами!`, () => {
                    galleryForm.reset();
                    selectedFiles = [];
                    renderImagePreviews();
                }, true);
            } else {
                let errorMsg = result.error || "Произошла неизвестная ошибка при создании галереи.";
                if (result.errors && Array.isArray(result.errors) && result.errors.length > 0) {
                    errorMsg += "\nНе удалось загрузить некоторые файлы:\n" + result.errors.map(e => e.message || e).join('\n');
                }
                showCustomModal(errorMsg, null, true);
                errorMessageDiv.textContent = errorMsg;
            }
        } catch (err) {
            console.error("🔴 Ошибка загрузки:", err);
            showCustomModal('Ошибка при создании галереи: ' + err.message, null, true);
            errorMessageDiv.textContent = 'Ошибка при создании галереи: ' + err.message;
        }
    });
}