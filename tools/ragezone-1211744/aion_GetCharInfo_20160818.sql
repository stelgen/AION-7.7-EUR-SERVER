USE [AionWorld_110]
GO
/****** Object:  StoredProcedure [dbo].[aion_GetCharInfo_20160818]    Script Date: 2023/3/12 21:04:16 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO

ALTER PROCEDURE [dbo].[aion_GetCharInfo_20160818]
	-- Add the parameters for the stored procedure here
	@nCharID	int
AS
BEGIN
	-- SET NOCOUNT ON added to prevent extra result sets from
	-- interfering with SELECT statements.
	SET NOCOUNT ON
	-- Insert statements for procedure here
	SELECT user_id, account_id, user_data.race, class, gender, 
	head_face_color, head_hair_color, head_eye_color, head_lip_color, head_face_type, head_hair_type, height_scale, head_voice_type, head_feat_type1, head_feat_type2,
	ISNULL(head_bump_type, 0), ISNULL(head_expression_type, 0),
	name_id, ISNULL(guild.id, 0) as guild_id, guild_rank, ISNULL(guild.name, '') as guild_name,  recreate_guild_time,
	org_server, cur_server,
	world, world_map_number, xlocation, ylocation, zlocation, dir, 
	last_normal_world, last_normal_xlocation, last_normal_ylocation, last_normal_zlocation, last_normal_dir,
	death_count, temporary_lost_exp,
	resurrect_world, resurrect_xlocation, resurrect_ylocation, resurrect_zlocation,
	builder, now_hit, now_mana, now_flight,
	exp, abyss_point, abyss_point_from_user, lev, stigmaPoint,
	cur_title_id,
	guild_intro, guild_nickname, 
	dbo.GetUnixtimeWithUTCAdjust(last_logout_time, dbo.GetUtcAdjustSecWithUTC_Local(GetUTCDate(), GetDate())) as last_logout_time,
	petition_msg,
	inventory_growth,
	char_warehouse_growth,
	0 as acc_warehouse_growth,
	delete_date,
	playtime,
	daily_comment,
	today_compare_time, today_abyss_kill_cnt, today_abyss_point,
	this_week_compare_time, this_week_abyss_kill_cnt, this_week_abyss_point,
	last_week_abyss_kill_cnt, last_week_abyss_point, total_abyss_kill_cnt, best_abyss_rank, is_freefly,
	feat_version,
	feat_face_shape,
	feat_forehead_shape,
	feat_eye_position,
	feat_eye_glabella,
	feat_eye_length,
	feat_eye_height,
	feat_eye_shape,
	feat_eye_tail,
	feat_eyeblow_pos,
	feat_eyeblow_angle,
	feat_eyeblow_shape,
	feat_nose_pos,
	feat_nose_bridge,
	feat_nose_side,
	feat_nose_tip,
	feat_cheek_shape,
	feat_mouth_pos,
	feat_mouth_size,
	feat_lip_thickness,
	feat_lip_tail,
	feat_lip_shape,
	feat_jaw_pos,
	feat_jaw_shape,
	feat_ear_shape,
	feat_head_size,
	feat_neck_thickness,
	feat_neck_length,
	feat_shoulder_size,
	feat_upper_size,
	feat_bust_size,
	feat_waist_size,
	feat_hip_size,
	feat_arm_thickness,
	feat_hand_size,
	feat_leg_thickness,
	feat_foot_size,
	feat_face_ratio,
	feat_wing_size,
	feat_arm_length,
	feat_leg_length,
	feat_shoulder_width,
	ISNULL(feat_head_figure, 0),
	optionflags,
	cashitem_inventory_growth,
	cashitem_warehouse_growth,
	accused_count,
	last_accuse_time,
	bot_point,
	vital_point,
	pvp_exp,
	serial_kill_point,
	serial_kill_penalty_duration,
	serial_kill_penalty_skill_rank,
	enhanced_stigma_slot_cnt,
	housing_id,
	fatigue_resttime_online,
	next_hotspot_use_time,
	gotcha_fever_point,
	gotcha_fever_expire_time,
	gotcha_fever_hit_count,
	last_explicit_beginner_force,
	absolute_exp,
	serial_guard_point, serial_guard_last_scantime,
	guild_offline_change_flag,
	isnull(head_eye_type, 0),
	isnull(head_dark_tail, 1),
	isnull(head_eye_color2, 0), 
	isnull(head_eye_lash, 0),
	isnull(feat_head_eye_size, 0), 
	isnull(feat_upper_height, 0),
	isnull(feat_arm_lower_thickness, 0),
	isnull(feat_hand_length, 0),
	isnull(feat_leg_lower_thickness, 0),
	isnull(absolute_fexp,0),
	is_jumping_character,
	absolute_ap
	FROM user_data LEFT OUTER JOIN guild ON user_data.guild_id = guild.id  
WHERE char_id = @nCharId

	SET NOCOUNT OFF
END

