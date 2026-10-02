
USE [AionWorld_110]
GO
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO

CREATE PROCEDURE [dbo].[aion_SetItemMatterOption]
	@itemId		bigint,
	@stat_enchant0	bigint,
	@stat_enchant1	bigint,
	@stat_enchant2	bigint,
	@stat_enchant3	bigint,
	@stat_enchant4	bigint,
	@stat_enchant5	bigint
AS
BEGIN
	SET NOCOUNT ON;
	update user_item_option set stat_enchant_name0 = @stat_enchant0, stat_enchant_name1 = @stat_enchant1, stat_enchant_name2 = @stat_enchant2, stat_enchant_name3 = @stat_enchant3, stat_enchant_name4 = @stat_enchant4, stat_enchant_name5 = @stat_enchant5 where id=@itemId
END
GO
