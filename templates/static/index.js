 function showMessageModal(message, callback) {
    const modal = document.getElementById('messageModal');
    const modalMessage = document.getElementById('messageModalText');
    const closeBtn = document.getElementById('closeMessageModal');
    const okBtn = document.getElementById('messageModalOkBtn');

    modalMessage.textContent = message;
    modal.style.display = 'flex';

    const closeHandler = function () {
    modal.style.display = 'none';
    if (callback && typeof callback === 'function') {
    callback();
}
    closeBtn.removeEventListener('click', closeHandler);
    okBtn.removeEventListener('click', closeHandler);
    window.removeEventListener('click', windowClickHandler);
};

    const windowClickHandler = function (event) {
    if (event.target === modal) {
    closeHandler();
}
};

    closeBtn.addEventListener('click', closeHandler);
    okBtn.addEventListener('click', closeHandler);
    window.addEventListener('click', windowClickHandler);
}

    function closeMessageModal() {
    const modal = document.getElementById('messageModal');
    if (modal) {
    modal.style.display = 'none';
}
}


    const searchInput = document.querySelector('.search-bar');
    const galleryRoot = document.getElementById("gallery-root");
    const loadingSpinner = document.getElementById("loading-spinner");

    let currentGalleryImages = [];
    let slideIndex = 0;

    let telegramUserID = null;
    let globalInitData = null;

    function getNullStringValue(nullStringObject) {
    return (nullStringObject && nullStringObject.Valid) ? nullStringObject.String : '';
}

    function escapeHtml(text) {
    if (text === null || text === undefined) {
    return '';
}
    text = String(text);
    const map = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
};
    return text.replace(/[&<>"']/g, function(m) { return map[m]; });
}



    document.addEventListener('DOMContentLoaded', function() {
    const tg = window.Telegram && window.Telegram.WebApp;

    if (loadingSpinner) {
    loadingSpinner.classList.remove('hidden');
    galleryRoot.innerHTML = '';
}

    if (tg) {
    tg.ready();
    globalInitData = tg.initData;
    console.log("Telegram WebApp:", tg);
    console.log("initData:", tg.initData);

    fetch('/auth', {
    method: 'POST',
    headers: {
    'Content-Type': 'application/json',
},
    body: JSON.stringify({ initData: globalInitData })
})
    .then(res => {
    if (!res.ok) {
    return res.json().then(errorData => {
    throw new Error(errorData.error || `HTTP error! status: ${res.status}`);
}).catch(() => {
    throw new Error(`HTTP error! status: ${res.status}`);
});
}
    return res.json();
})
    .then(data => {
    if (data.ok) {
    telegramUserID = data.user.id;
    console.log("✅ Telegram user authenticated:", data.user);
    const urlParams = new URLSearchParams(window.location.search);
    const query = urlParams.get("q");
    if (query) {
    searchInput.value = query;
}
    loadGalleries(query);
} else {
    showMessageModal("Ошибка авторизации через Telegram: " + (data.error || 'Неизвестная ошибка'), () => {
    galleryRoot.innerHTML = `<p class="no-galleries">Ошибка авторизации. Галереи не загружены.</p>`;
    if (loadingSpinner) {
    loadingSpinner.classList.add('hidden');
}
});
}
})
    .catch(error => {
    showMessageModal("Ошибка запроса к /auth: " + error.message, () => {
    galleryRoot.innerHTML = `<p class="no-galleries">Ошибка подключения к серверу. Галереи не загружены.</p>`;
    if (loadingSpinner) {
    loadingSpinner.classList.add('hidden');
}
});
});
} else {
    console.warn("Telegram WebApp is not available. Running in a standard browser environment.");
    showMessageModal("Это приложение предназначено для запуска в Telegram. Авторизация недоступна.", () => {
    galleryRoot.innerHTML = `<p class="no-galleries">Приложение запущено вне Telegram. Галереи не могут быть загружены без авторизации.</p>`;
    if (loadingSpinner) {
    loadingSpinner.classList.add('hidden');
}
});
}
});

    searchInput.addEventListener('keydown', function (event) {
    if (event.key === 'Enter') {
    const q = this.value.trim();
    loadGalleries(q);
}
});

    async function loadGalleries(q = "") {
    if (loadingSpinner) {
    loadingSpinner.classList.remove('hidden');
    galleryRoot.innerHTML = '';
}

    try {
    if (!globalInitData) {
    console.warn("globalInitData отсутствует. Галереи будут загружены без учета избранного статуса.");
}

    const headers = {
    'Content-Type': 'application/json'
};
    if (globalInitData) {
    headers['X-Telegram-Init-Data'] = globalInitData;
}

    const response = await fetch(`/api/galleries?q=${encodeURIComponent(q)}`, {
    method: 'GET',
    headers: headers
});

    if (!response.ok) {
    const errorData = await response.json().catch(() => ({error: `HTTP status ${response.status}`}));
    throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
}
    const data = await response.json();
    console.log("Данные от сервера /api/galleries:", data);

    if (data.ok) {
    renderGalleries(data.galleries);
} else {
    galleryRoot.innerHTML = `<p class="no-galleries">Ошибка при получении галерей: ${escapeHtml(data.error || 'Неизвестная ошибка')}</p>`;
}
} catch (err) {
    console.error("Ошибка загрузки галерей:", err);
    showMessageModal("Ошибка сервера при загрузке галерей: " + escapeHtml(err.message || 'Неизвестная ошибка'));
    galleryRoot.innerHTML = `<p class="no-galleries">Ошибка сервера при загрузке галерей.</p>`;
} finally {
    if (loadingSpinner) {
    loadingSpinner.classList.add('hidden');
}
}
}

    function searchByTag(element, event) {
    event.stopPropagation();
    const tag = element.dataset.tag;
    searchInput.value = tag;
    loadGalleries(tag);
}

    function renderGalleries(galleries) {
    console.log("Данные галерей для рендеринга:", galleries);

    if (!galleries || galleries.length === 0) {
    galleryRoot.innerHTML = `<p class="no-galleries">Галереи не найдены.</p>`;
    return;
}

    let html = `<div class="gallery-masonry-grid">`;
    galleries.forEach(gal => {
    let previewImageSrc = gal.previewURL;
    if (previewImageSrc && previewImageSrc !== '/static/no-image-placeholder.png') {
    if (previewImageSrc.startsWith('/')) {
    previewImageSrc = previewImageSrc.substring(1);
}
    previewImageSrc = `/secured_gallery_images/${previewImageSrc}`;
} else {
    previewImageSrc = '/static/no-image-placeholder.png';
}

    const imageCount = gal.imageCount || 0;
    const imageAlt = imageCount > 0 ? `Превью галереи ${escapeHtml(gal.name)}` : 'Нет изображений';

    const creatorFirstName = getNullStringValue(gal.creatorFirstName);
    const creatorLastName = getNullStringValue(gal.creatorLastName);
    const creatorUsername = getNullStringValue(gal.creatorUsername);
    let creatorPhotoURL = getNullStringValue(gal.creatorPhotoURL);

    if (!creatorPhotoURL) {
    creatorPhotoURL = '/static/default_avatar.png';
} else {
    if (!creatorPhotoURL.startsWith('http://') && !creatorPhotoURL.startsWith('https://') && !creatorPhotoURL.startsWith('data:')) {
    if (creatorPhotoURL.startsWith('/')) {
    creatorPhotoURL = creatorPhotoURL.substring(1);
}
    creatorPhotoURL = `/secured_gallery_images/${creatorPhotoURL}`;
}
}


    let displayName = creatorFirstName;
    if (creatorLastName) {
    displayName += ` ${creatorLastName}`;
}
    if (!displayName && creatorUsername) {
    displayName = `@${creatorUsername}`;
}
    if (!displayName) {
    displayName = 'Неизвестный';
}

    const isFavoriteClass = gal.isFavorite ? 'active' : '';
    const favoriteIcon = gal.isFavorite ? '&#9733;' : '&#9734;';

    html += `
        <div class="gallery-item">
            <h2>${escapeHtml(gal.name)}</h2>
            <div class="gallery-preview" data-gallery-id="${gal.id}">
                <img src="${escapeHtml(previewImageSrc)}" alt="${escapeHtml(imageAlt)}" class="gallery-preview-image" loading="lazy">
            </div>
            <div class="gallery-details">
                <div class="tags">
                    ${gal.tags?.length ?
    `<p><span class="hash-symbol">#:</span> ${gal.tags.map(tag => `<span class="tag clickable-tag" data-tag="${escapeHtml(tag)}" onclick="searchByTag(this, event)">${escapeHtml(tag)}</span>`).join(' ')}</p>` :
    `<p class="no-tags"></p>`
}
                </div>
                <div class="gallery-user-info-wrapper">
                    <div class="gallery-user-info" data-creator-id="${gal.creatorID}"> <img src="${escapeHtml(creatorPhotoURL)}" alt="${escapeHtml(displayName)}" class="user-avatar" loading="lazy">
                        <p class="user-name">${escapeHtml(displayName)}</p>
                    </div>
                    <button class="favorite-button ${isFavoriteClass}" data-gallery-id="${gal.id}" onclick="toggleFavorite(this, event)">
                        ${favoriteIcon}
                    </button>
                </div>
            </div>
        </div>`;
});
    html += `</div>`;
    galleryRoot.innerHTML = html;

    document.querySelectorAll('.gallery-preview').forEach(preview => {
    preview.addEventListener('click', function(event) {
    if (event.target.classList.contains('gallery-preview-image') || event.target.classList.contains('gallery-preview')) {
    const galleryID = this.dataset.galleryId;
    if (galleryID) {
    openGallerySlider(galleryID);
}
}
});
});
    document.querySelectorAll('.gallery-user-info').forEach(userInfo => {
    userInfo.addEventListener('click', function(event) {
    const creatorID = this.dataset.creatorId;
    if (creatorID) {
    viewUserGalleries(creatorID, event);
}
});
});
}

    async function toggleFavorite(button, event) {
    event.stopPropagation();
    if (!telegramUserID || !globalInitData) {
    showMessageModal("Для добавления в избранное необходимо авторизоваться.");
    return;
}

    const galleryID = button.dataset.galleryId;
    if (!galleryID) {
    console.error("Не найден ID галереи для кнопки избранного.");
    showMessageModal("Ошибка: Не удалось определить галерею.");
    return;
}

    const isCurrentlyFavorite = button.classList.contains('active');
    const method = isCurrentlyFavorite ? 'DELETE' : 'POST';
    const endpoint = `/api/favorites/${galleryID}`;

    try {
    const response = await fetch(endpoint, {
    method: method,
    headers: {
    'X-Telegram-Init-Data': globalInitData,
    'Content-Type': 'application/json'
}
});

    if (!response.ok) {
    const errorData = await response.json().catch(() => ({error: `HTTP status ${response.status}`}));
    throw new Error(errorData.error || `Сервер ответил со статусом ${response.status}`);
}

    const data = await response.json();
    if (data.ok) {
    if (isCurrentlyFavorite) {
    button.classList.remove('active');
    button.innerHTML = '&#9734;';
} else {
    button.classList.add('active');
    button.innerHTML = '&#9733;';
}
} else {
    showMessageModal("Ошибка при изменении статуса избранного: " + (data.error || 'Неизвестная ошибка'));
    console.error("Ошибка API favorites:", data.error);
}
} catch (error) {
    console.error("Ошибка при запросе избранного:", error);
    showMessageModal("Ошибка сети при изменении избранного: " + error.message);
}
}


    function viewUserGalleries(userID, event) {
    if (userID) {
    window.location.href = `/user_galleries?user_id=${userID}`;
} else {
    showMessageModal("Ошибка: ID пользователя недоступен.");
}
}

    async function openGallerySlider(galleryID, startIndex = 0) {
    const modalImage = document.getElementById("modalImage");
    const captionText = document.getElementById("caption");
    const myModal = document.getElementById("myModal");

    myModal.style.display = "flex";
    modalImage.src = 'static/loading.gif';
    modalImage.classList.add('loading-gif-modal');
    captionText.innerHTML = 'Загрузка изображений...';

    try {
    const response = await fetch(`/api/gallery_images/${galleryID}`);
    if (!response.ok) {
    const errorData = await response.json().catch(() => ({error: `HTTP status ${response.status}`}));
    throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
}
    const data = await response.json();

    if (data.ok) {
    currentGalleryImages = data.images;
} else {
    showMessageModal("Ошибка при получении изображений галереи: " + escapeHtml(data.error || 'Неизвестная ошибка'));
    console.error("❌ Ошибка получения изображений:", data.error);
    currentGalleryImages = [];
}

} catch (e) {
    console.error("Ошибка при загрузке изображений галереи:", e);
    showMessageModal("Ошибка загрузки изображений галереи. Попробуйте еще раз.");
    currentGalleryImages = [];
} finally {
    modalImage.classList.remove('loading-gif-modal');
}

    if (currentGalleryImages.length === 0) {
    showMessageModal("В этой галерее нет изображений.");
    closeModal();
    return;
}

    slideIndex = startIndex;
    showSlides(slideIndex);
}

    function showSlides(n) {
    const modalImage = document.getElementById("modalImage");
    const captionText = document.getElementById("caption");

    if (currentGalleryImages.length === 0) {
    modalImage.src = '';
    captionText.innerHTML = 'Нет изображений в галерее.';
    return;
}

    if (n >= currentGalleryImages.length) {
    slideIndex = 0;
}
    if (n < 0) {
    slideIndex = currentGalleryImages.length - 1;
}

    modalImage.src = currentGalleryImages[slideIndex];
    captionText.innerHTML = `Изображение ${slideIndex + 1} из ${currentGalleryImages.length}`;
}

    function plusSlides(n) {
    showSlides(slideIndex += n);
}

    function closeModal() {
    document.getElementById("myModal").style.display = "none";
    currentGalleryImages = [];
    slideIndex = 0;
    const modalImage = document.getElementById("modalImage");
    modalImage.classList.remove('loading-gif-modal');
}

    window.onclick = function (event) {
    const modal = document.getElementById("myModal");
    if (event.target === modal) {
    closeModal();
}
};

    function createGallery() {
    if (telegramUserID) {
    window.location.href = `/create_gallery`;
} else {
    showMessageModal("Ошибка: Telegram ID не получен. Авторизация не пройдена.");
}
}

    function viewMyGalleries() {
    if (telegramUserID) {
    window.location.href = '/my_galleries';
} else {
    showMessageModal("Ошибка: Telegram ID не получен. Авторизация не пройдена.");
}
}

    function goToHomePage() {
    window.location.href = `/`;
}
    function viewMySubscriptions() {
    if (telegramUserID) {
    window.location.href = `/my_subscriptions`;
} else {
    showMessageModal("Для просмотра подписок необходимо авторизоваться.");
}
}
    function viewFavoriteGalleries() {
    if (telegramUserID) {
    window.location.href = `/favorite_galleries`;
} else {
    showMessageModal("Ошибка: Telegram ID не получен. Авторизация не пройдена.");
}
}
