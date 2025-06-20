-- Таблица users
CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     telegram_user_id BIGINT UNIQUE NOT NULL,
                                     telegram_username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    photo_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, -- Рекомендую WITH TIME ZONE
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP  -- Рекомендую WITH TIME ZONE
                             );
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_telegram_user_id ON users (telegram_user_id);


-- Таблица galleries
CREATE TABLE IF NOT EXISTS galleries (
                                         id BIGSERIAL PRIMARY KEY,
                                         name VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL, -- Это telegram_user_id из таблицы users
    folder_path VARCHAR(255) UNIQUE NOT NULL,
    image_count INTEGER DEFAULT 0,
    preview_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                             CONSTRAINT fk_galleries_user
                             FOREIGN KEY (user_id)
    REFERENCES users(telegram_user_id) -- Ссылка на telegram_user_id, а не на id users
                         ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_galleries_user_id ON galleries (user_id);
CREATE INDEX IF NOT EXISTS idx_galleries_name ON galleries (name);


-- Таблица gallery_images (ИСПРАВЛЕНО: id на BIGSERIAL)
CREATE TABLE IF NOT EXISTS gallery_images (
                                              id BIGSERIAL PRIMARY KEY, -- ИЗМЕНЕНО
                                              gallery_id BIGINT NOT NULL,
                                              full_size_image_path TEXT NOT NULL,
                                              preview_image_path TEXT,
                                              created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                                              CONSTRAINT fk_gallery_images_gallery
                                              FOREIGN KEY (gallery_id)
    REFERENCES galleries(id)
    ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_gallery_images_gallery_id ON gallery_images (gallery_id);


-- Таблица tags (ИСПРАВЛЕНО: id на BIGSERIAL)
CREATE TABLE IF NOT EXISTS tags (
                                    id BIGSERIAL PRIMARY KEY, -- ИЗМЕНЕНО
                                    gallery_id BIGINT NOT NULL,
                                    tag VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                             CONSTRAINT fk_tags_gallery
                             FOREIGN KEY (gallery_id)
    REFERENCES galleries(id)
                         ON DELETE CASCADE,
    UNIQUE(gallery_id, tag)
    );
CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags (tag);
CREATE INDEX IF NOT EXISTS idx_tags_gallery_id ON tags (gallery_id);


-- Таблица favorites
CREATE TABLE IF NOT EXISTS favorites (
                                         user_id BIGINT NOT NULL,
                                         gallery_id BIGINT NOT NULL,
                                         created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                                         PRIMARY KEY (user_id, gallery_id), -- Составной первичный ключ

    CONSTRAINT fk_favorites_user
    FOREIGN KEY (user_id)
    REFERENCES users(telegram_user_id) -- Ссылка на telegram_user_id
    ON DELETE CASCADE,
    CONSTRAINT fk_favorites_gallery
    FOREIGN KEY (gallery_id)
    REFERENCES galleries(id)
    ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites (user_id);
CREATE INDEX IF NOT EXISTS idx_favorites_gallery_id ON favorites (gallery_id);


-- Таблица subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
                                             subscriber_id BIGINT NOT NULL,
                                             target_user_id BIGINT NOT NULL,
                                             created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                                             PRIMARY KEY (subscriber_id, target_user_id), -- Составной первичный ключ

    CONSTRAINT fk_subscriptions_subscriber
    FOREIGN KEY (subscriber_id)
    REFERENCES users(telegram_user_id) -- Ссылка на telegram_user_id
    ON DELETE CASCADE,
    CONSTRAINT fk_subscriptions_target_user
    FOREIGN KEY (target_user_id)
    REFERENCES users(telegram_user_id) -- Ссылка на telegram_user_id
    ON DELETE CASCADE,
    CONSTRAINT chk_no_self_subscribe
    CHECK (subscriber_id <> target_user_id) -- Проверка, что пользователь не может подписаться на себя
    );
CREATE INDEX IF NOT EXISTS idx_subscriptions_subscriber_id ON subscriptions (subscriber_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_target_user_id ON subscriptions (target_user_id);