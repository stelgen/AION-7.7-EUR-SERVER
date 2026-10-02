SET NOCOUNT ON;
-- FIX 4 (#180 Angry Catster): aion_SetItemMatterOption — сохранение манастоунов/заточек статов
-- (исходник: RaGEZONE att 270537, БД AionWorld_110 → адаптировано под _AionWorldNew114_rc)
PRINT '=== FIX 4: aion_SetItemMatterOption ===';
USE [_AionWorldNew114_rc];
IF OBJECT_ID('dbo.user_item_option') IS NULL
  PRINT 'WARN: table user_item_option отсутствует — фикс неприменим';
ELSE
BEGIN
  IF OBJECT_ID('dbo.aion_SetItemMatterOption','P') IS NULL
  BEGIN
    EXEC(N'CREATE PROCEDURE [dbo].[aion_SetItemMatterOption]
      @itemId bigint, @stat_enchant0 bigint, @stat_enchant1 bigint, @stat_enchant2 bigint,
      @stat_enchant3 bigint, @stat_enchant4 bigint, @stat_enchant5 bigint
    AS BEGIN
      SET NOCOUNT ON;
      UPDATE user_item_option SET
        stat_enchant_name0 = @stat_enchant0, stat_enchant_name1 = @stat_enchant1,
        stat_enchant_name2 = @stat_enchant2, stat_enchant_name3 = @stat_enchant3,
        stat_enchant_name4 = @stat_enchant4, stat_enchant_name5 = @stat_enchant5
      WHERE id = @itemId;
    END');
    PRINT 'proc aion_SetItemMatterOption created';
  END
  ELSE
    PRINT 'proc aion_SetItemMatterOption exists';
END
GO
-- FIX 5 (community bugs.txt): индекс IX_user_item_sealed_char_id на user_item_sealed
PRINT '=== FIX 5: IX_user_item_sealed_char_id ===';
USE [_AionWorldNew114_rc];
IF COL_LENGTH('dbo.user_item_sealed','char_id') IS NULL
  PRINT 'WARN: нет колонки char_id';
ELSE IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name='IX_user_item_sealed_char_id' AND object_id=OBJECT_ID('dbo.user_item_sealed'))
BEGIN
  CREATE NONCLUSTERED INDEX IX_user_item_sealed_char_id ON dbo.user_item_sealed (char_id);
  PRINT 'index created';
END
ELSE
  PRINT 'index already exists';
GO
-- FIX 6 (community bugs.txt): заглушка Log_TblGameWorldInfo_UpdateMainStatus в Aion_log
PRINT '=== FIX 6: Log_TblGameWorldInfo_UpdateMainStatus ===';
USE [Aion_log];
IF OBJECT_ID('dbo.Log_TblGameWorldInfo_UpdateMainStatus','P') IS NULL
BEGIN
  EXEC(N'CREATE PROCEDURE [dbo].[Log_TblGameWorldInfo_UpdateMainStatus] AS BEGIN SET NOCOUNT ON; END');
  PRINT 'stub created (0 параметров; при ошибке сигнатуры см. LogServer .err)';
END
ELSE
  PRINT 'stub exists';
GO
-- Итоговая сверка недостающих процедур из community-списка bugs.txt
PRINT '=== AUDIT (bugs.txt vs наш БД) ===';
USE [_AionWorldNew114_rc];
SELECT name FROM sys.procedures
WHERE name IN ('aion_GetItemCollectionLevelList','aion_GetItemCollectionList',
 'aion_GetItemCollectionExpiredList','aion_GetItemCollectionCompleteTimeLimitList',
 'aion_GetItemCollectionCompleteList','aion_LoadReinventInfo','aion_LoadFameInfo');
IF @@ROWCOUNT = 0 PRINT '-> все ещё отсутствуют (заглушки по сигнатурам из наших логов, roadmap Этап 1)';
PRINT '=== DONE ===';
