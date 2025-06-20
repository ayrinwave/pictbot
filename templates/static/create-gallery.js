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

function showCustomModal(message, callback) {
    modalMessage.textContent = message;
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
    customModal.style.display = 'none';
}

function openImageModal(src) {
    if (!imageViewerModal || !modalImage) {
        console.error('Modal elements for image viewer not found.');
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
    errorMessageDiv.textContent = '';
}

function renderImagePreviews() {
    imagePreviewsContainer.innerHTML = '';

    if (selectedFiles.length === 0) {
        imagePreviewsContainer.style.display = 'none';
        return;
    } else {
        imagePreviewsContainer.style.display = 'grid';
    }

    selectedFiles.forEach((file, originalIndex) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const previewItem = document.createElement('div');
            previewItem.classList.add('preview-item');

            const img = document.createElement('img');
            img.src = e.target.result;
            img.alt = `Preview ${file.name}`;
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
        };
        reader.readAsDataURL(file);
    });
}

galleryImagesInput.addEventListener('change', (event) => {
    clearErrorMessage();

    const newFiles = Array.from(event.target.files);

    if (newFiles.length === 0) {
        galleryImagesInput.value = "";
        return;
    }

    let filesToAdd = [];
    let rejectedCount = 0;

    for (const file of newFiles) {
        const isDuplicate = selectedFiles.some(
            (existingFile) => existingFile.name === file.name && existingFile.size === file.size
        );

        if (isDuplicate) {
            console.warn(`Файл ${file.name} уже выбран и будет проигнорирован.`);
            rejectedCount++;
            continue;
        }

        if (selectedFiles.length + filesToAdd.length >= 10) {
            showCustomModal("Вы можете загрузить не более 10 файлов! Удалите одно из выбранных, чтобы добавить новое.");
            rejectedCount++;
            break;
        }

        if (!file.type.startsWith('image/')) {
            showCustomModal(`Файл ${file.name} не является изображением и будет проигнорирован.`);
            rejectedCount++;
            continue;
        }
        if (file.size > 32 * 1024 * 1024) { // 32 MB
            showCustomModal(`Файл ${file.name} превышает 32MB!`);
            rejectedCount++;
            continue;
        }

        filesToAdd.push(file);
    }

    selectedFiles = selectedFiles.concat(filesToAdd);

    renderImagePreviews();

    galleryImagesInput.value = "";

    if (rejectedCount > 0) {
        showCustomModal(`Некоторые файлы были проигнорированы из-за размера, типа или дубликатов. Успешно добавлено: ${filesToAdd.length}`);
    }
});

document.addEventListener('DOMContentLoaded', function() {
    const tg = window.Telegram.WebApp;

    if (tg) {
        tg.ready();
        console.log("Telegram WebApp is initialized.");
        console.log("initData:", tg.initData);

        let telegramUserID = null;
        let telegramUsername = null;

        fetch('/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ initData: tg.initData })
        })
            .then(res => res.json())
            .then(data => {
                if (data.ok) {
                    telegramUserID = data.user.id;
                    telegramUsername = data.user.username || '';

                    document.getElementById('telegramUserID').value = telegramUserID;
                    document.getElementById('telegramUsername').value = telegramUsername;

                    console.log('✅ Telegram user authenticated:', data.user);
                    submitButton.disabled = false;
                } else {
                    showCustomModal("Ошибка авторизации через Telegram: " + (data.error || 'Неизвестная ошибка'), () => {});
                    submitButton.disabled = true;
                    console.error('❌ Ошибка авторизации:', data.error);
                }
            })
            .catch(error => {
                showCustomModal("Ошибка запроса к /auth: " + error.message, () => {});
                submitButton.disabled = true;
                console.error('❌ Ошибка запроса к /auth:', error);
            });
    } else {
        console.warn("Telegram WebApp is not available. Running in a standard browser environment.");
        showCustomModal("Это приложение предназначено для запуска в Telegram. Авторизация недоступна.", () => {});
        submitButton.disabled = true;
    }
});

// Обработчик отправки формы
galleryForm.addEventListener('submit', async function (event) {
    event.preventDefault(); // Предотвращаем стандартную отправку формы

    clearErrorMessage();

    if (submitButton.disabled) {
        showCustomModal("Форма заблокирована из-за проблем с авторизацией.");
        return;
    }

    let telegramUserID = document.getElementById('telegramUserID').value;
    if (!telegramUserID || telegramUserID === 'null' || telegramUserID === '') {
        showCustomModal("Ошибка: Telegram ID не получен, невозможно создать галерею. Пожалуйста, попробуйте перезапустить приложение в Telegram.");
        return;
    }

    const galleryName = document.getElementById('galleryName').value.trim();
    if (!galleryName) {
        showCustomModal("Пожалуйста, введите название галереи.");
        return;
    }

    if (selectedFiles.length === 0) {
        showCustomModal("Пожалуйста, выберите хотя бы одно изображение.");
        return; // Прекращаем отправку формы
    }

    const tagsInput = document.getElementById('tagsInput').value.trim();
    const tagsArray = tagsInput.split(',').map(tag => tag.trim()).filter(tag => tag !== '');

    const formData = new FormData();
    formData.append('galleryName', galleryName);
    tagsArray.forEach(tag => {
        formData.append("tagsInput", tagsInput);
    });
    selectedFiles.forEach((file) => {
        formData.append('galleryImages', file);
    });

    if (window.Telegram && window.Telegram.WebApp && window.Telegram.WebApp.initData) {
        formData.append('initData', window.Telegram.WebApp.initData);
    } else {
        console.error("Telegram WebApp initData не доступен при отправке формы!");
        showCustomModal("Ошибка: Нет данных авторизации Telegram. Пожалуйста, перезапустите приложение.", () => {});
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
            throw new Error(`Ошибка: ${result}`);
        }

        if (response.ok && result.ok) {
            showCustomModal(`Галерея "${result.galleryName}" успешно создана с ${result.imageCount} файлами!`, () => {
                galleryForm.reset();
                selectedFiles = [];
                renderImagePreviews();
            });
        } else {
            let errorMsg = result.error || "Произошла неизвестная ошибка при создании галереи.";
            if (result.errors && Array.isArray(result.errors) && result.errors.length > 0) {
                errorMsg += "\nНе удалось загрузить некоторые файлы:\n" + result.errors.map(e => e.message || e).join('\n');
            }
            showCustomModal(errorMsg);
            errorMessageDiv.textContent = errorMsg;
        }
    } catch (err) {
        console.error("🔴 Ошибка загрузки:", err);
        showCustomModal('Ошибка при создании галереи: ' + err.message);
        errorMessageDiv.textContent = 'Ошибка при создании галереи: ' + err.message;
    }
});