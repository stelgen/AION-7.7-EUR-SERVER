-- ============================================================================
-- REF58 Log-процедуры, ОТСУТСТВУЮЩИЕ в прод Aion_log (2026-10-05)
-- Кандидаты на деплой: LogServer64 может вызывать их так же, как
-- Log_TblGameWorldInfo_InitializeCount (error 2812 спам).
-- Статус: НЕ ДЕПЛОЕНЫ, ждут решения. Тела 1-в-1 из REF58_aion_log (кит 5.8).
-- ============================================================================

-- 1) Обновляет total-строку (по семантике имени - вероятно ZONE_ID=0/total_connection)
CREATE proc [dbo].[Log_TblGameWorldInfo_UpdateTotalMainStatus]
	@light_users			int,
	@dark_users				int,
	@npc_count				int,
	@pc_store_light_users	int,
	@world_id				tinyint,
	@zone_id				int
AS
	UPDATE	aiongm_ur.TBL_GAME_WORLD_INFO
	SET		LIGHT_USERS = @light_users
			, DARK_USERS = @dark_users
			, NPC_COUNT = @npc_count
			, PC_STORE_LIGHT_USERS = @pc_store_light_users
			, REGDATE = GETDATE()
	WHERE	WORLD_ID = @world_id and ZONE_ID = @zone_id
GO

-- 2) Вставляет запись мониторинга сервера (CPU/память) в TBL_WORLD_STATUS_INFO
CREATE proc [dbo].[Log_TblWorldStatusInfo_InsertServerinfo]
	@world_id			int,
	@server_nm			int,
	@cpu_usage			int,
	@free_phy_memory	nvarchar(20),
	@process_memory		nvarchar(20)
AS
	INSERT aiongm_ur.TBL_WORLD_STATUS_INFO (WORLD_ID, SERVER_NM, CONCURRENT_USERS, CPU_USAGE, FREE_PHY_MEMORY, PROCESS_MEMORY, DB_FREE, REGDATE)
	VALUES (@world_id, @server_nm, 1, @cpu_usage, @free_phy_memory, @process_memory, 1, GETDATE())
GO

-- Откат (если будут деплоены): DROP PROCEDURE dbo.Log_TblGameWorldInfo_UpdateTotalMainStatus; DROP PROCEDURE dbo.Log_TblWorldStatusInfo_InsertServerinfo;
