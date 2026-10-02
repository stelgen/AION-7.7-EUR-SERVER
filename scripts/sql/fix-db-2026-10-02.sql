-- =============================================================
-- AION 7.7 PTS EU — DB fixes 2026-10-02 (VM 109 AION-77-PTS-EU)
-- Устраняет 3 класса ошибок из CacheServer\log\DBReport_Alert.DBErr
-- и лога LogServer64. Идемпотентно: можно запускать повторно.
-- Источник: лог-скан 02.10.2026; docs/fixes-known-issues.md
-- Запуск: sqlcmd -S localhost -U sa -P 123 -i fix-db-2026-10-02.sql
-- =============================================================
SET NOCOUNT ON;

-- FIX 1: aion_GetDeletedCharList падал с
-- "Index 'IX_delete_complete_date' on table 'user_data' does not exist"
-- (процедура зовёт user_data with(nolock, index=IX_delete_complete_date))
PRINT '=== FIX 1: index IX_delete_complete_date ===';
USE [_AionWorldNew114_rc];
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_delete_complete_date' AND object_id = OBJECT_ID('dbo.user_data'))
BEGIN
  CREATE NONCLUSTERED INDEX IX_delete_complete_date ON dbo.user_data (delete_complete_date, delete_date)
    INCLUDE (char_id, user_id, account_id, account_name, guild_id, guild_rank);
  PRINT 'index created';
END
ELSE
  PRINT 'index already exists';
GO

-- FIX 2: aion_ProcessWithdrawAccount ссылался на несуществующий linked server
-- "Could not find server 'RC-AIONAUTHDB' in sys.servers"
-- и на БД aionaccdeldb (dbo.del_account: колонки seq/account_id/stat —
-- ровно те, что использует процедура; выведено из её текста).
PRINT '=== FIX 2: linked server RC-AIONAUTHDB ===';
IF NOT EXISTS (SELECT 1 FROM sys.servers WHERE name = 'RC-AIONAUTHDB')
BEGIN
  EXEC master.dbo.sp_addlinkedserver @server = N'RC-AIONAUTHDB', @srvproduct = N'',
       @provider = N'SQLNCLI11', @datasrc = N'localhost', @catalog = N'AionAccounts';
  PRINT 'linked server created';
END
ELSE
  PRINT 'linked server exists';
EXEC master.dbo.sp_addlinkedsrvlogin @rmtsrvname = N'RC-AIONAUTHDB', @useself = N'FALSE',
     @locallogin = NULL, @rmtuser = N'sa', @rmtpassword = N'123';
PRINT 'linked login mapped';
GO
IF DB_ID('aionaccdeldb') IS NULL
BEGIN
  DECLARE @cdb nvarchar(max) = N'CREATE DATABASE [aionaccdeldb] COLLATE Korean_Wansung_CI_AS';
  EXEC (@cdb);
  PRINT 'DB aionaccdeldb created (Korean_Wansung_CI_AS)';
END
ELSE
  PRINT 'DB aionaccdeldb exists';
GO
USE [aionaccdeldb];
IF OBJECT_ID('dbo.del_account') IS NULL
BEGIN
  CREATE TABLE dbo.del_account (
    seq        int NOT NULL,
    account_id int NOT NULL,
    stat       int NOT NULL CONSTRAINT df_del_account_stat DEFAULT (0),
    CONSTRAINT pk_del_account PRIMARY KEY (seq)
  );
  PRINT 'table del_account created (seq, account_id, stat)';
END
ELSE
  PRINT 'table del_account exists';
GO

-- Проверка починенных процедур
PRINT '=== TEST procs ===';
USE [_AionWorldNew114_rc];
BEGIN TRY
  EXEC dbo.aion_GetDeletedCharList 1, 1790972000;
  PRINT 'aion_GetDeletedCharList OK';
END TRY
BEGIN CATCH
  PRINT 'aion_GetDeletedCharList ERR';
  PRINT ERROR_MESSAGE();
END CATCH
GO
BEGIN TRY
  EXEC dbo.aion_ProcessWithdrawAccount 1790972000, 1;
  PRINT 'aion_ProcessWithdrawAccount OK';
END TRY
BEGIN CATCH
  PRINT 'aion_ProcessWithdrawAccount ERR';
  PRINT ERROR_MESSAGE();
END CATCH
GO

-- FIX 3: LogServer64 падал на отсутствующей процедуре
-- "Could not find stored procedure 'Log_TblGameServerInfo_UpdateServerstatus'"
-- Вызов из LogServer64: {call Log_TblGameServerInfo_UpdateServerstatus(1,1,5)}
-- (worldId, status, serverType). Заглушка: обновляет таблицу, если она есть.
PRINT '=== FIX 3: Log_TblGameServerInfo_UpdateServerstatus (Aion_log) ===';
USE [Aion_log];
IF OBJECT_ID('dbo.Log_TblGameServerInfo_UpdateServerstatus','P') IS NULL
BEGIN
  DECLARE @s nvarchar(max) = N'CREATE PROCEDURE [dbo].[Log_TblGameServerInfo_UpdateServerstatus] @worldId int, @status int, @serverType int AS BEGIN SET NOCOUNT ON; IF OBJECT_ID(''dbo.Log_TblGameServerInfo'') IS NOT NULL UPDATE dbo.Log_TblGameServerInfo SET status = @status WHERE worldId = @worldId AND serverType = @serverType; END';
  EXEC (@s);
  PRINT 'proc created';
END
ELSE
  PRINT 'proc exists';
BEGIN TRY
  EXEC dbo.Log_TblGameServerInfo_UpdateServerstatus 1, 1, 5;
  PRINT 'Log_TblGameServerInfo_UpdateServerstatus OK';
END TRY
BEGIN CATCH
  PRINT 'stub ERR';
  PRINT ERROR_MESSAGE();
END CATCH
PRINT '=== DONE ===';
