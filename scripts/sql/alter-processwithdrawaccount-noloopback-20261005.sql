-- ============================================================================
-- aion_ProcessWithdrawAccount: REMOVE LOOPBACK LINKED SERVER (2026-10-05)
-- ROOT CAUSE ночи 04-05.10.2026 (4 краша CacheD64.exe, AV 0xc0000005
-- @+0x18c31f = "Intentional exception" после CheckIOThreadDeadlock):
--   CacheD-таймер вызывает эту процу; внутри BEGIN TRAN (XACT_ABORT ON) +
--   3 ссылки на [RC-AIONAUTHDB].aionaccdeldb.dbo.del_account — LOOPBACK
--   linked server на тот же инстанс. Компиляция/исполнение с loopback-
--   ссылкой внутри активной транзакции зависает (command=NOP, 100% CPU,
--   self-deadlock compile-семафора с вложенной loopback-сессией) ->
--   IOThread CacheD занят вечно -> Super-Lag (181-193с) -> суицид CacheD
--   -> Server64 умирает следом -> мир падает.
-- ФИКС: прямая cross-database ссылка aionaccdeldb.dbo.del_account
--   (тот же инстанс, linked server не нужен, DTC не нужен).
-- БЕКАПЫ перед изменением:
--   D:\_REF58\prod-backups\AionWorld-20261005-prealter.bak (15 МБ compressed)
--   D:\_REF58\prod-backups\Aion_log-20261005-preinitcount.bak
-- Верификация: definition без RC-AIONAUTHDB; smoke EXEC @nDeleteTime=0,
--   @nServerId=1 — мгновенно; compile queue = 0.
-- РОЛЛБАК: выполнить ORIG-тело с CREATE->ALTER (ORIG сохранён на VM
--   C:\Temp\proc_aion_ProcessWithdrawAccount_ORIGINAL_20261005.sql).
-- ============================================================================

-- ===================== НОВАЯ ВЕРСИЯ (применена) ============================
ALTER procedure [dbo].[aion_ProcessWithdrawAccount]

             @nDeleteTime     int,

             @nServerId                      int

as

set nocount on

set XACT_ABORT ON

BEGIN TRAN

declare @lastMaxSeq int

declare @newMaxSeq int

declare @notDeletedMaxSeq int

set @lastMaxSeq = -1

set @newMaxSeq = 0

select @lastMaxSeq = int_value from server_info where server_id = @nServerId and info_name = 'SERVER_PROCESS_WITHDRAW_ACCOUNT_SEQ'

if (@lastMaxSeq = -1)
begin
             insert server_info (server_id, info_name, int_value, int64_value, str_value) values(@nServerId, 'SERVER_PROCESS_WITHDRAW_ACCOUNT_SEQ', 0, 0, '')

             set @lastMaxSeq = 0
end

set @notDeletedMaxSeq = @lastMaxSeq

select @notDeletedMaxSeq = MIN(seq) from user_data

join aionaccdeldb.dbo.del_account as del_account

on user_data.account_id = del_account.account_id

where

user_data.delete_date = 0

and del_account.stat = 2

if (@notDeletedMaxSeq < @lastMaxSeq)
begin
             set @lastMaxSeq = @notDeletedMaxSeq
end

select @newMaxSeq = MAX(seq) from aionaccdeldb.dbo.del_account

if (@lastMaxSeq < @newMaxSeq)
begin
             UPDATE user_data

             SET user_data.delete_date = @nDeleteTime

             from user_data

             join aionaccdeldb.dbo.del_account as del_account

             on user_data.account_id = del_account.account_id

             where

             user_data.delete_date = 0

             and

             del_account.stat = 2

             and

             del_account.seq >= @lastMaxSeq

             and

             del_account.seq <= @newMaxSeq

             update server_info set int_value = @newMaxSeq, int64_value = @lastMaxSeq, str_value = CONVERT(nvarchar(255), GETDATE(), 121) where server_id = @nServerId and info_name = 'SERVER_PROCESS_WITHDRAW_ACCOUNT_SEQ'
end

COMMIT

set nocount off
GO

-- ===================== ОРИГИНАЛ (для роллбака) =============================
-- Отличие НОВОЙ от ОРИГИНАЛА: только отсутствие префикса [RC-AIONAUTHDB].
-- у 3 ссылок на aionaccdeldb.dbo.del_account. Оригинал 1-в-1 сохранён на VM:
-- C:\Temp\proc_aion_ProcessWithdrawAccount_ORIGINAL_20261005.sql
-- Роллбак: взять оригинал, заменить CREATE procedure -> ALTER procedure,
-- выполнить. ПРИМЕЧАНИЕ: после роллбака краши CacheD вернутся.
