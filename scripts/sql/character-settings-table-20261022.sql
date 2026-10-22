-- ============================================================================
-- Т30: Таблица настроек клиента (zlib-XML страницы per-charId)
-- CacheD хранит клиентские настройки (options/quickbar/favorites) как zlib XML.
-- Наш aion-cached использует эту таблицу для персиста.
-- ============================================================================
CREATE TABLE dbo.character_settings (
	char_id		int NOT NULL,
	page		tinyint NOT NULL,
	settings_data	varbinary(max) NOT NULL,
	data_len	int NOT NULL DEFAULT 0,
	updated_at	datetime NOT NULL DEFAULT GETDATE(),
	PRIMARY KEY (char_id, page)
);
GO

-- Гранты для aionop_ro (наш aion-cached)
GRANT SELECT, INSERT, UPDATE, DELETE ON dbo.character_settings TO aionop_ro;
GO

-- Откат: DROP TABLE dbo.character_settings;
