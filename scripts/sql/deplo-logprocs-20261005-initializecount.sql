-- ============================================================================
-- Log_TblGameWorldInfo_InitializeCount - DEPLOY (2026-10-05)
-- Источник: REF58_aion_log (кит 5.8), тело восстановлено 1-в-1.
-- Причина: LogServer64 вызывает эту процедуру при старте мира, но в прод-БД
--          Aion_log она ОТСУТСТВОВАЛА -> error 2812 спам.
--          Главная находка ночи 04-05.10: отсутствие части Log_Tbl*-проц +
--          max server memory 2048 (пустой план-кэш) -> compile storm ->
--          RESOURCE_SEMAPHORE_QUERY_COMPILE очередь на минуты ->
--          CacheD64 IOThread hang 181-193с -> CheckIOThreadDeadlock ->
--          "Intentional exception" -> AV-краш CacheD64.exe @+0x18c31f (x3).
-- Бекап перед деплоем: D:\_REF58\prod-backups\Aion_log-20261005-preinitcount.bak
-- Верификация: EXEC @world_id=1 OK, счётчики зон мира 1 обнулены.
-- РОЛЛБАК: DROP PROCEDURE dbo.Log_TblGameWorldInfo_InitializeCount;
--          (или RESTORE DATABASE Aion_log из бекапа выше)
-- ============================================================================

CREATE PROCEDURE [dbo].[Log_TblGameWorldInfo_InitializeCount]
	@world_id	tinyint
AS
	UPDATE	aiongm_ur.TBL_GAME_WORLD_INFO
	SET		LIGHT_USERS = 0, DARK_USERS = 0, NPC_COUNT = 0, PC_STORE_LIGHT_USERS = 0, PC_STORE_DARK_USERS = 0, REGDATE = GETDATE()
	WHERE	WORLD_ID = @world_id
GO
