USE [lin2db]
GO
/****** Object:  Table [dbo].[userno]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[userno](
	[uid] [int] NULL
) ON [PRIMARY]
GO
/****** Object:  Table [dbo].[user_time]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_time](
	[uid] [int] NOT NULL,
	[account] [varchar](14) NOT NULL,
	[present_time] [int] NOT NULL,
	[next_time] [int] NULL,
	[total_time] [int] NOT NULL,
	[op_date] [datetime] NOT NULL,
	[flag] [tinyint] NOT NULL,
 CONSTRAINT [PK_user_time] PRIMARY KEY CLUSTERED 
(
	[uid] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_stat]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_stat](
	[uid] [int] NOT NULL,
	[country] [varchar](4) NOT NULL,
	[gender] [int] NOT NULL,
	[birthyear] [varchar](4) NOT NULL,
 CONSTRAINT [PK_user_stat] PRIMARY KEY CLUSTERED 
(
	[uid] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_pay]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[user_pay](
	[start_date] [datetime] NOT NULL,
	[account] [nchar](14) NOT NULL,
	[end_date] [datetime] NOT NULL,
	[uid] [bigint] NOT NULL
) ON [PRIMARY]
GO
/****** Object:  Table [dbo].[user_mmoweb_banned_ip]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_mmoweb_banned_ip](
	[id] [int] IDENTITY(1,1) NOT NULL,
	[ip] [varchar](60) NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_mmoweb_banned_hwid]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_mmoweb_banned_hwid](
	[id] [int] IDENTITY(1,1) NOT NULL,
	[fg] [varchar](35) NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_mmoweb_banned_ga]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_mmoweb_banned_ga](
	[id] [int] IDENTITY(1,1) NOT NULL,
	[ga] [varchar](35) NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_info]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_info](
	[account] [varchar](14) NOT NULL,
	[create_date] [datetime] NOT NULL,
	[ssn] [varchar](50) NOT NULL,
	[status_flag] [int] NOT NULL,
	[kind] [int] NOT NULL,
	[userkind] [smallint] NOT NULL,
	[userintro] [varchar](255) NULL,
	[service_flag] [int] NOT NULL,
	[card_number] [int] NULL,
	[recommId] [varchar](14) NULL,
	[email] [varchar](50) NULL,
	[code] [varchar](15) NULL,
 CONSTRAINT [PK_user_info] PRIMARY KEY CLUSTERED 
(
	[account] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_etc]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_etc](
	[account] [varchar](14) NOT NULL,
	[accountCustomizeBitSet] [varbinary](2048) NULL,
 CONSTRAINT [PK_user_etc] PRIMARY KEY CLUSTERED 
(
	[account] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_data]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[user_data](
	[uid] [bigint] NOT NULL,
	[user_char_num] [bigint] NOT NULL,
	[world] [nvarchar](20) NULL,
	[char_name] [nvarchar](50) NOT NULL,
	[char_id] [int] NOT NULL,
	[account_name] [nvarchar](50) NOT NULL,
	[Lev] [tinyint] NOT NULL,
	[create_date] [datetime] NOT NULL,
	[use_time] [int] NULL,
	[subjob0_class] [int] NOT NULL,
	[subjob1_class] [int] NOT NULL,
	[subjob2_class] [int] NOT NULL,
	[subjob3_class] [int] NOT NULL,
 CONSTRAINT [PK_user_data] PRIMARY KEY CLUSTERED 
(
	[uid] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
/****** Object:  Table [dbo].[user_count]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[user_count](
	[record_time] [datetime] NOT NULL,
	[server_id] [tinyint] NOT NULL,
	[world_user] [int] NOT NULL,
	[limit_user] [int] NOT NULL,
	[auth_user] [int] NOT NULL,
	[wait_user] [int] NOT NULL,
	[dayofweek] [int] NOT NULL
) ON [PRIMARY]
GO
/****** Object:  Table [dbo].[user_block]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_block](
	[BlockDate] [varchar](8) NOT NULL,
	[account] [varchar](14) NOT NULL,
 CONSTRAINT [PK_user_block] PRIMARY KEY CLUSTERED 
(
	[account] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_auth]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_auth](
	[account] [varchar](24) NOT NULL,
	[password] [binary](16) NOT NULL,
	[quiz1] [varchar](255) NOT NULL,
	[quiz2] [varchar](255) NULL,
	[answer1] [binary](32) NOT NULL,
	[answer2] [binary](32) NOT NULL,
	[new_pwd_flag] [tinyint] NULL,
	[otp_flag] [tinyint] NULL,
	[md5password] [varchar](40) NULL,
 CONSTRAINT [PK_user_auth] PRIMARY KEY CLUSTERED 
(
	[account] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_account_link]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_account_link](
	[accold] [varchar](14) NULL,
	[accnew] [varchar](14) NULL,
	[prefix] [varchar](14) NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[user_account]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[user_account](
	[uid] [int] IDENTITY(1,1) NOT NULL,
	[account] [varchar](14) NOT NULL,
	[pay_stat] [smallint] NOT NULL,
	[login_flag] [int] NOT NULL,
	[warn_flag] [int] NOT NULL,
	[block_flag] [int] NOT NULL,
	[block_flag2] [int] NOT NULL,
	[last_login] [datetime] NULL,
	[last_logout] [datetime] NULL,
	[subscription_flag] [int] NOT NULL,
	[last_game] [int] NULL,
	[last_world] [int] NULL,
	[last_ip] [varchar](15) NULL,
	[block_end_date] [datetime] NULL,
	[forbidden_servers] [binary](16) NULL,
	[premium_expire] [datetime] NOT NULL,
	[email] [varchar](50) NULL,
	[allow_ip] [varchar](255) NULL,
	[hkey] [varchar](16) NULL,
	[cheat] [int] NOT NULL,
	[mask] [varchar](255) NULL,
	[special_gates] [int] NOT NULL,
	[donate] [int] NOT NULL,
 CONSTRAINT [PK_user_account] PRIMARY KEY CLUSTERED 
(
	[account] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[st_acc_daily]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[st_acc_daily](
	[date] [char](10) NOT NULL,
	[Total_acc] [int] NULL,
	[New_acc] [int] NULL,
	[Max_con] [int] NULL,
	[Max_con_time] [char](16) NULL,
	[Active_acc] [int] NULL,
	[Avg_con] [int] NULL,
	[Max01] [int] NULL,
	[Max01_time] [char](16) NULL,
	[Max02] [int] NULL,
	[Max02_time] [char](16) NULL,
	[Min_con] [int] NULL,
	[Min_con_time] [char](16) NULL,
	[Login_count] [int] NULL,
	[Account_count] [int] NULL,
	[Avg_playing_time] [bigint] NULL,
	[Total_playing_time] [bigint] NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[st_acc_active]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[st_acc_active](
	[date] [char](8) NOT NULL,
	[Total_acc] [int] NULL,
	[Never_login] [int] NULL,
	[Active_login] [int] NULL,
	[Active_90] [int] NULL,
	[Active_60] [int] NULL,
	[Active_30] [int] NULL,
	[Active_15] [int] NULL,
	[Active_7] [int] NULL,
	[Active_3] [int] NULL,
	[Active_1] [int] NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[ssn]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[ssn](
	[ssn] [char](13) NOT NULL,
	[name] [varchar](15) NOT NULL,
	[email] [varchar](50) NOT NULL,
	[newsletter] [tinyint] NOT NULL,
	[job] [int] NOT NULL,
	[phone] [varchar](16) NOT NULL,
	[mobile] [varchar](20) NULL,
	[reg_date] [datetime] NOT NULL,
	[zip] [varchar](6) NOT NULL,
	[addr_main] [varchar](255) NOT NULL,
	[addr_etc] [varchar](255) NOT NULL,
	[account_num] [tinyint] NOT NULL,
	[status_flag] [int] NOT NULL,
	[final_news_date] [datetime] NULL,
	[master] [varchar](14) NULL,
	[valid_email_date] [datetime] NULL,
	[final_master_date] [datetime] NOT NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[server]    Script Date: 01/22/2023 15:56:49 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[server](
	[id] [nvarchar](255) NULL,
	[name] [nvarchar](255) NULL,
	[ip] [nvarchar](255) NULL,
	[inner_ip] [nvarchar](255) NULL,
	[ageLimit] [nvarchar](255) NULL,
	[pk_flag] [nvarchar](255) NULL,
	[kind] [nvarchar](255) NULL,
	[port] [nvarchar](255) NULL,
	[region] [nvarchar](255) NULL,
	[master_id] [nvarchar](255) NULL
) ON [PRIMARY]
GO
/****** Object:  StoredProcedure [dbo].[ap_LogoutWithPoint]    Script Date: 01/22/2023 15:56:50 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[ap_LogoutWithPoint]
@block_end_date datetime,
@last_login datetime,
@last_logout datetime
AS
SELECT 0
GO
/****** Object:  StoredProcedure [dbo].[ap_LoginWithPoint]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[ap_LoginWithPoint]
@block_end_date datetime,
@last_login datetime,
@last_logout datetime
AS
SELECT 0
GO
/****** Object:  Table [dbo].[block_reason_code]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[block_reason_code](
	[block_reason] [int] NOT NULL,
	[block_desc] [varchar](50) NOT NULL,
	[flag] [tinyint] NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[block_msg]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[block_msg](
	[uid] [int] NULL,
	[account] [varchar](14) NOT NULL,
	[msg] [varchar](50) NULL,
	[reason] [int] NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[banancheg]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE TABLE [dbo].[banancheg](
	[uid] [bigint] NOT NULL,
	[cid] [bigint] NOT NULL,
	[bid] [bigint] NOT NULL,
	[date] [datetime] NOT NULL,
	[kicked] [bit] NOT NULL
) ON [PRIMARY]
GO
/****** Object:  Table [dbo].[hauthd_log]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[hauthd_log](
	[time] [datetime] NOT NULL,
	[account] [varchar](14) NOT NULL,
	[ip] [varchar](15) NOT NULL,
	[hkey] [varchar](16) NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[gm_illegal_login]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[gm_illegal_login](
	[account] [varchar](15) NOT NULL,
	[try_date] [datetime] NOT NULL,
	[ip] [varchar](15) NOT NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[item_code]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[item_code](
	[item_id] [int] NOT NULL,
	[name] [varchar](20) NOT NULL,
	[duration] [int] NOT NULL,
	[active_date] [datetime] NOT NULL,
 CONSTRAINT [PK_item_code] PRIMARY KEY CLUSTERED 
(
	[item_id] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[mw_accounts_utm]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[mw_accounts_utm](
	[id] [int] IDENTITY(1,1) NOT NULL,
	[master_id] [int] NULL,
	[account] [varchar](30) NULL,
	[email] [varchar](150) NULL,
	[affiliate] [int] NULL,
	[promo_id] [int] NULL,
	[utm_master_reg_count] [int] NULL,
	[utm_master_reg_date] [datetime] NULL,
	[utm_iso] [varchar](3) NULL,
	[utm_mw] [varchar](150) NULL,
	[fg] [varchar](35) NULL,
	[ga] [varchar](35) NULL,
	[ip] [varchar](60) NULL,
	[date_create] [datetime] NOT NULL
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  Table [dbo].[worldstatus]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
SET ANSI_PADDING ON
GO
CREATE TABLE [dbo].[worldstatus](
	[idx] [int] NOT NULL,
	[server] [varchar](50) NOT NULL,
	[status] [tinyint] NOT NULL,
 CONSTRAINT [PK__worldstatus__00551192] PRIMARY KEY CLUSTERED 
(
	[idx] ASC
)WITH (PAD_INDEX  = OFF, STATISTICS_NORECOMPUTE  = OFF, IGNORE_DUP_KEY = OFF, ALLOW_ROW_LOCKS  = ON, ALLOW_PAGE_LOCKS  = ON) ON [PRIMARY]
) ON [PRIMARY]
GO
SET ANSI_PADDING OFF
GO
/****** Object:  StoredProcedure [dbo].[web_CreateAccount]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[web_CreateAccount] 
	@account varchar(14),
	@email varchar(50),
	@password binary(100)
AS

SELECT account FROM user_account WHERE account = @account
IF @@ROWCOUNT <> 0 
RETURN 2

BEGIN TRAN
	
	INSERT INTO user_account (account, pay_stat, login_flag, warn_flag, block_flag, block_flag2, last_game, last_world)
		VALUES (@account, 1001, 0, 0, 0, 0, 8, 0)  
	IF @@ERROR <> 0 GOTO DO_ROLLBACK

	INSERT INTO user_info(account,ssn,kind,email) VALUES (@account,777, 99, @email)
	IF @@ERROR <> 0 GOTO DO_ROLLBACK

	INSERT INTO user_auth(account,password,quiz1,quiz2,answer1,answer2,new_pwd_flag,otp_flag)
		VALUES (@account, CONVERT(binary, @password), '', '', CONVERT(binary, ''), CONVERT(binary, ''),0,0)	
	IF @@ERROR <> 0 GOTO DO_ROLLBACK

	INSERT INTO user_etc(account,accountCustomizeBitSet) VALUES (@account, CONVERT(binary, ''))
	IF @@ERROR <> 0 GOTO DO_ROLLBACK
	
COMMIT TRAN
RETURN 1

DO_ROLLBACK:
ROLLBACK TRAN
RETURN 0
GO
/****** Object:  StoredProcedure [dbo].[l2p_TempCreateAccount]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS OFF
GO
SET QUOTED_IDENTIFIER OFF
GO
CREATE PROCEDURE [dbo].[l2p_TempCreateAccount]
@account varchar(14),
@ssn varchar(13)
AS
-- 歹固 拌沥阑 积己窍扁 困茄 风凭涝聪促. 
-- 捞固 拌沥捞 积己登绢 乐绰 林刮殿废锅龋狼 版快俊绰 积己登瘤 臼嚼聪促. 
-- 努肺令海鸥扁埃俊父 荤侩登绢具 钦聪促. 

DECLARE @account_num int

SELECT account FROM user_info WHERE account = @account
IF @@ROWCOUNT <> 0 
BEGIN
	print 'Already Exist'
	RETURN
END

SELECT @account_num= account_num FROM ssn WHERE ssn =@ssn
If @@rowcount  =  0
begin
	set @account_num = 1
end
else
begin
	set @account_num = @account_num + 1
end

BEGIN TRAN	
	IF @account_num = 1
		Insert ssn ( ssn, name, email, newsletter, job, phone, mobile, reg_date, zip, addr_main, addr_etc, account_num, status_flag )
		values (@ssn, '抛胶飘拌沥', 'newjisu@ncsoft.net',0,0,'02-1234-1234','011-1234-1234',getdate(),'','','',@account_num,0)		
	ELSE
		UPDATE ssn SET account_num = @account_num WHERE ssn =  @ssn
	IF @@ERROR <> 0 GOTO DO_ROLLBACK
	INSERT INTO user_account (account, pay_stat) VALUES (@account, 0)
	IF @@ERROR <> 0 GOTO DO_ROLLBACK
	Insert user_auth ( account, password, quiz1, quiz2, answer1, answer2 ) 
	values ( @account, 0xB53AA65D7C98EF3F0A93B5B578E2C4C4, '郴啊 促囱 檬殿切背 捞抚篮 公均老鳖?', '郴啊 促囱 檬殿切背 捞抚篮 公均老鳖?', 0x93A5EFCC45DA1D96A33A1C1CD14B6D6D, 0x93A5EFCC45DA1D96A33A1C1CD14B6D6D)
	IF @@ERROR <> 0 GOTO DO_ROLLBACK
	Insert user_info ( account, create_date, ssn, status_flag, kind )
	values ( @account, getdate(),@ssn, 0, 99  )
	IF @@ERROR <> 0 GOTO DO_ROLLBACK	
	
	
	--update user_account set pay_stat=101, login_flag=0 where account = @account
commit TRAN
RETURN 1

DO_ROLLBACK:
ROLLBACK TRAN
RETURN 0
GO
/****** Object:  StoredProcedure [dbo].[hauthd_login]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[hauthd_login]
@uid int,
@ip varchar(15),
@hkey varchar(16)
AS
SELECT pay_stat AS ok FROM user_account WITH (nolock) WHERE uid = @uid
GO
/****** Object:  StoredProcedure [dbo].[CCU_Stat]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
--exec Pr_acc_act
--select * from st_acc_active

create         procedure [dbo].[CCU_Stat]
as
begin
 
 --select count(account) Accounts from user_account 
  
 select T.Da as Date, T.Ti as Time,
       T.world_user as Total, A.world_user as Aria, B.world_user Blackbird
 from 
(select convert(varchar, record_time, 111) Da,
        substring(convert(varchar, record_time, 100),13,7) Ti, world_user
 from user_count
 where record_time > '2008-01-22 11:00:00.000' and server_id = 0 ) T,
(select convert(varchar, record_time, 111) Da,
       substring(convert(varchar, record_time, 100),13,7) Ti, world_user
 from user_count
 where record_time > '2008-01-22 11:00:00.000' and server_id = 25 ) A,
(select convert(varchar, record_time, 111) Da,
        substring(convert(varchar, record_time, 100),13,7) Ti, world_user
 from user_count
 where record_time > '2008-01-22 11:00:00.000' and server_id = 45 ) B
where T.Da = A.Da and T.Da = B.Da and T.Ti = A.Ti and T.Ti = B.Ti 
order by T.Da   , T.Ti  

end
GO
/****** Object:  StoredProcedure [dbo].[ap_SUserTime]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE      PROCEDURE [dbo].[ap_SUserTime] 
@useTime 	int, 
@uid 		int,
@payStat 	int = 2200,	-- ? parameter? ???? ?? ?? ??? ???? ? SP? ?????.
				-- ???? default? ??? ????? ??. 2200? ? ?? 2? ?? ?? ?? ??? ??.
@loginTime 	datetime = 0	-- ? parameter? ???? ?? ??? ?? ???? ????.
				-- ? ?? ??? ????? ??.
AS

DECLARE @payType 	tinyint
DECLARE @SPECIFICDATE	tinyint
DECLARE @SPECIFICTIME	tinyint

SET @SPECIFICTIME = 2
SET @SPECIFICDATE = 5
SET @payType = (@payStat % 1000) / 100

IF @payType = @SPECIFICTIME		-- ??
BEGIN
	UPDATE	user_time 
	SET 	total_time 	= total_time - @useTime, 
		present_time 	= present_time - @useTime
	WHERE	uid = @uid
END
ELSE IF @payType = @SPECIFICDATE	-- ?? ??
BEGIN
--	SELECT	TOP 1 UP.start_date	
	--FROM	user_pay UP inner join user_account UA on (UP.account = UA.account)  
	--WHERE	UA.uid= @uid
	--AND	UP.start_date < @loginTime

	IF (@@ROWCOUNT > 0)	-- ?? ?? ??? ??? ?? ?? ?? ?????
	BEGIN
		UPDATE	user_time 
		SET 	total_time 	= total_time - @useTime, 
			present_time 	= present_time - @useTime
		WHERE	uid = @uid		
	END
END
GO
/****** Object:  StoredProcedure [dbo].[ap_SUserData]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**  
*Purpose            : »ç¿ëÀÚ ºÎ°¡ Á¤º¸¸¦ ±â·ÏÇÑ´Ù.   
*Record Set         : ¾øÀ½  
*Error Code         : ¾øÀ½  
*History            : 2010-07-29 kundong ÃÖÃÊ ÀÛ¼º  
*/  
  
CREATE PROCEDURE [dbo].[ap_SUserData]  
  @account   VARCHAR (14)  -- °ÔÀÓ °èÁ¤   
, @userdata_size SMALLINT   -- »ç¿ëÀÚ ºÎ°¡ Á¤º¸ Å©±â  
, @userdata   VARBINARY(2048)  -- »ç¿ëÀÚ ºÎ°¡ Á¤º¸  
AS  
  
DECLARE @destAccount VARCHAR (14)  
  
SELECT @destAccount = account FROM user_etc WITH(NOLOCK)  
WHERE account = @account  
  
IF @@ROWCOUNT = 0  
BEGIN  
 INSERT user_etc (account, accountCustomizeBitSet)   
 VALUES (@account, @userdata)  
END  
ELSE  
BEGIN  
 UPDATE user_etc  
 SET  accountCustomizeBitSet = @userdata  
 WHERE account = @account  
END
GO
/****** Object:  StoredProcedure [dbo].[ap_SNewPwd]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
create PROCEDURE [dbo].[ap_SNewPwd]  
@account varchar(14), 
@pwd binary(16),
@encFlag tinyint
AS
UPDATE	user_auth 
set		new_pwd_flag = @encFlag,
		password = @pwd  
WHERE	account = @account
GO
/****** Object:  StoredProcedure [dbo].[ap_SLog]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE  PROCEDURE [dbo].[ap_SLog] 
@uid int, @lastlogin datetime, @lastlogout datetime, @LastGame int, @LastWorld tinyint, @LastIP varchar(15)
AS
UPDATE user_account 
SET last_login = @lastlogin, last_logout=@lastlogout, last_world=@lastWorld, last_game=@lastGame, last_ip=@lastIP
WHERE uid=@uid
GO
/****** Object:  StoredProcedure [dbo].[ap_SetServerStatus]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_SetServerStatus]
*Purpose	:   Update the state of the server
*E-Mail		:   veni@ncsoft.net
*Date		:   09/19/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameServerNo
*		gameServerNo
*	@statusCode
*		the current state of the game server
*/

create	PROCEDURE [dbo].[ap_SetServerStatus]
	@gameServerNo	TINYINT,
	@statusCode		TINYINT			
AS

IF @gameServerNo = 0 
BEGIN
	UPDATE	worldStatus
	SET		status = @statusCode
END
ELSE
BEGIN
	UPDATE	worldStatus
	SET		status = @statusCode
	WHERE	idx = @gameServerNo
END

RETURN


SET QUOTED_IDENTIFIER ON
GO
/****** Object:  StoredProcedure [dbo].[ap_SetPasswordResetFlag]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_SetPasswordResetFlag]
*Purpose	:   Set the flag to force the user change the password before logging in.
*E-Mail		:   veni@ncsoft.net
*Date		:   09/19/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameAccountNo
*		gameAccountNo
*
*/

create	PROCEDURE [dbo].[ap_SetPasswordResetFlag]
@gameAccountNo	INT
AS

UPDATE	user_account 
SET		login_flag = (login_flag | 1) 
WHERE	uid = @gameAccountNo
GO
/****** Object:  StoredProcedure [dbo].[ap_SetIllegalLoginTrace]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_SetIllegalLoginTrace]
*Purpose	:   Record the trace log for illegal login try.
*E-Mail		:   veni@ncsoft.net
*Date		:   09/19/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameAccountNo
*		gameAccountNo
*	@ip
*		IP address of the host from which the user tried to login.
*	@traceTypeCode
*		trace type code
*/

create	PROCEDURE [dbo].[ap_SetIllegalLoginTrace]
@gameAccountNo	INT,
@IP				VARCHAR(15),
@traceTypeCode	SMALLINT
AS

INSERT gm_illegal_login ( account, ip ) 
VALUES (@gameAccountNo, @IP)
GO
/****** Object:  StoredProcedure [dbo].[ap_SetGameRestriction]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	: Minae, Kim
*SP Name	: dbo.ap_SetGameRestriction
*E-Mail		: mingii@ncsoft.net
*Date		: 06/25/2008
*Input Parameter    :   
*	@uid
*	@flag
*
*Output Parameter   :   
*
*Modification Memo	:
*/
create PROCEDURE [dbo].[ap_SetGameRestriction]
	@uid	INT,
	@flag	TINYINT
AS

IF @flag = 1
BEGIN
	UPDATE user_account SET login_flag = login_flag | 256 WHERE uid = @uid
END
ELSE
BEGIN
	UPDATE user_account SET login_flag = (login_flag  | 256) ^ 256 WHERE uid = @uid
END

IF @@ROWCOUNT = 0
BEGIN
	RETURN 1
END

RETURN 0
GO
/****** Object:  StoredProcedure [dbo].[ap_SetGameBlock]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	: Seunghoon, Park
*SP Name	: dbo.ap_SetGameBlock
*E-Mail		: surym@ncsoft.net
*Date		: 02/24/2009
*Input Parameter    :   
*	@uid
*	@block_reason	block_flag : Reference::block_reason_code table (0 ? as a blocking clear))
*
*Output Parameter   :   
*
*	0 - success
*	1 - failure
*
*Modification Memo	:	2009.02.04, surym@ncsoft.net, creates
*						
*/
CREATE  PROCEDURE [dbo].[ap_SetGameBlock]
	@uid			INT,
	@blockFlag		INT
AS

UPDATE user_account SET block_flag = @blockFlag WHERE uid = @uid

IF @@ROWCOUNT = 0
BEGIN
	RETURN 1
END

RETURN 0
GO
/****** Object:  StoredProcedure [dbo].[ap_SetConcurrentUserStatistics]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
create PROCEDURE [dbo].[ap_SetConcurrentUserStatistics] 
@serverNo					SMALLINT,
@concurrentWorldUserCount	INT,
@concurrentUserLimit		INT,
@concurrentAuthUserCount	INT,
@concurrentAuthWaitCount	INT
	
AS

INSERT INTO user_count (server_id, world_user, limit_user, auth_user, wait_user)
VALUES (@serverNo, @concurrentWorldUserCount, @concurrentUserLimit, @concurrentAuthUserCount, @concurrentAuthWaitCount)
GO
/****** Object:  StoredProcedure [dbo].[ap_GUserTime]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[ap_GUserTime]  @uid int, @userTime int OUTPUT
AS
SELECT @userTime=total_time FROM user_time WITH (nolock) 
WHERE uid = @uid
GO
/****** Object:  StoredProcedure [dbo].[ap_GStatEtc]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[ap_GStatEtc]    
@account VARCHAR(14)    
AS    
  
DECLARE @accountCustomizeBitSet VARBINARY(2048)    
DECLARE @userDataLenth INT    
    
SET @accountCustomizeBitSet = 0x0    
SET @userDataLenth = 0    
    
SELECT @accountCustomizeBitSet = accountCustomizeBitSet, @userDataLenth = DATALENGTH( accountCustomizeBitSet )    
FROM dbo.user_etc WITH (NOLOCK)    
WHERE account = @account    
    
SELECT @accountCustomizeBitSet AS userData, @userDataLenth AS userDataLenth    
	FROM user_info UI WITH(NOLOCK)
	WHERE UI.account = @account
GO
/****** Object:  StoredProcedure [dbo].[ap_GStat]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE PROCEDURE [dbo].[ap_GStat]
@account varchar(14), 
@uid int OUTPUT, 
@payStat int OUTPUT, 
@loginFlag int OUTPUT, 
@warnFlag int OUTPUT, 
@blockFlag int OUTPUT, 
@blockFlag2 int OUTPUT, 
@subFlag int OUTPUT, 
@lastworld tinyint OUTPUT,
@block_end_date datetime OUTPUT
 AS
SELECT @uid=uid, 
	 @payStat=pay_stat,
              @loginFlag = login_flag, 
              @warnFlag = warn_flag, 
              @blockFlag = block_flag, 
              @blockFlag2 = block_flag2, 
              @subFlag = subscription_flag , 
              @lastworld=last_world, 
              @block_end_date=block_end_date 
               FROM user_account WITH (nolock)
WHERE account=@account
--maddaemon fix 08
UPDATE user_account set last_login=getdate() WHERE account=@account
--update user_account set block_flag2=0 where block_end_date<getdate() and account=@account
GO
/****** Object:  StoredProcedure [dbo].[ap_GPwdWithFlag]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
CREATE   PROCEDURE [dbo].[ap_GPwdWithFlag]  
@account varchar(14)
, @pwd binary(16) output
, @flag tinyint output
--, @otpflag tinyint = 0 output
AS
if(not exists(select account from user_auth where account=@account)) 
begin
if(@account not like '%[^a-zA-Z0-9]%') 
begin
SELECT	@pwd=0x00000000000000000000000000000000, @flag=3
--exec ap_AutoReg @account
end
end 
else 
begin
SELECT	@pwd=password 
	, @flag=new_pwd_flag
--	, @otpflag = otp_flag 
FROM	user_auth with (nolock) 
WHERE	account=@account
end
GO
/****** Object:  StoredProcedure [dbo].[ap_GPwd]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/*
ALTER  PROCEDURE [dbo].[ap_GPwd]  @account varchar(14), @pwd binary(16) output
AS
declare @ssn nvarchar(13)
set @ssn=substring(STR(FLOOR(RAND()*8999999+1000000)),4,7)+substring(STR(FLOOR(RAND()*899999+100000)),5,6)
INSERT INTO [ssn](ssn,name,email,job,phone,zip,addr_main,addr_etc,account_num) VALUES (@ssn,@account+'@kamael.ru','0',0,'telphone','123456','','',1)
INSERT INTO user_account (account,pay_stat) VALUES (@account, 1)
INSERT INTO user_auth (account,password,quiz1,quiz2,answer1,answer2,new_pwd_flag) VALUES (@account,0,0,0,0,0,0)
INSERT INTO user_info (account,ssn,kind,code) VALUES (@account,@ssn, 99, '123')
SELECT @pwd=password FROM user_auth with (nolock) WHERE account=@account
*/

CREATE PROCEDURE [dbo].[ap_GPwd]  @account varchar(14), @pwd binary(16) output
AS
SELECT @pwd=password FROM user_auth with (nolock) WHERE account=@account
GO
/****** Object:  StoredProcedure [dbo].[ap_GetSSN]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_GetSSN]
*Purpose	:   Get the ssn of the user
*E-Mail		:   veni@ncsoft.net
*Date		:   09/19/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameAccountNo
*		gameAccountNo
*
*/

create	PROCEDURE [dbo].[ap_GetSSN]
	@gameAccountNo	INT,
	@ssn CHAR(13) OUTPUT
AS

SET @ssn = ''

SELECT	@ssn = ssn 
FROM	user_info with (nolock) 
WHERE	account = @gameAccountNo
GO
/****** Object:  StoredProcedure [dbo].[ap_GetServers]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_GetServers]
*Purpose	:   Get information of the servers
*E-Mail		:   veni@ncsoft.net
*Date		:   09/19/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	None
*		
*/

create	PROCEDURE [dbo].[ap_GetServers]
AS

SELECT	id, name, ip, inner_ip, ageLimit, pk_flag, kind, port, region 
FROM	server 
ORDER BY id
GO
/****** Object:  StoredProcedure [dbo].[ap_GetRestriction]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_GetRestriction]
*Purpose	:   Get message for the restriction of the user
*E-Mail		:   veni@ncsoft.net
*Date		:   09/05/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameAccountNo
*		gameAccountNo
*
*/
create	PROCEDURE [dbo].[ap_GetRestriction]
@gameAccountNo	INT
AS

SELECT	reason, msg
FROM	block_msg with (nolock)
WHERE	uid = @gameAccountNo
GO
/****** Object:  StoredProcedure [dbo].[ap_GetGameAccountNo]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
---------------------------------------------
/**
*Creator	:   Jaewon Ryu
*SP Name	:   dbo.[ap_GetGameAccountNo]
*Purpose	:   Get gameAccountNo of the gameAccount
*E-Mail		:   veni@ncsoft.net
*Date		:   09/05/2008
*Usage		:
*M.History	:	
*===============================================================================
*Input Parameter    :   
*	@gameAccount
*		gameAccount
*
*/
create	PROCEDURE [dbo].[ap_GetGameAccountNo]
@gameAccount	VARCHAR(16)
AS

SELECT	uid
FROM	user_account with (nolock)
WHERE	account = @gameAccount
GO
/****** Object:  StoredProcedure [dbo].[Pr_acc_dailyreport]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
--exec Pr_acc_dailyreport
--select * from st_acc_daily

CREATE            procedure [dbo].[Pr_acc_dailyreport]
as
begin

declare @act1 int
declare @act2 int
declare @act3 int
declare @act4 int
declare @act5 int
declare @act6 int
declare @act7 int
declare @act8 int
declare @act9 int
declare @act10 int
declare @act11 int
declare @act12 int
declare @act13 int
declare @act14 int
declare @act15 int

declare @str1 char(16)
declare @str2 char(16)
declare @str3 char(16)
declare @str4 char(16)

-- Temp value --
set @act1 = 0
set @act2 = 0
set @act3 = 0
set @act4 = 0
set @act5 = 0
set @act6 = 0
set @act7 = 0
set @act8 = 0
set @act9 = 0
set @act10 = 0
set @act11 = 0
set @act12 = 0
set @act13 = 0
set @act14 = 0
set @act15 = 0

-- Total account count --
select  @act1=isnull(count(account), 0)
from user_info with (nolock)
where create_date < convert(varchar(10),getdate() ,20)

-- Daily new subscriber count --
select  @act2=isnull(count(account), 0)
from user_info with (nolock)
where create_date >=  convert(varchar(10),getdate()-1 ,20) and create_date < convert(varchar(10),getdate() ,20)
group by datepart(yy,create_date),datepart(m,create_date),datepart(d,create_date) 

-- Total game Max connected account --
select @act3=isnull(max(w.max_user), 0)
from (select datepart(yy,record_time) Year ,datepart(m,record_time) Month,datepart(d,record_time) Day, 
                    datepart(hh,record_time) Hour, datepart(n,record_time) Min, sum(world_user) max_user
          from user_count with (nolock)
          where record_time >= convert(varchar(10),getdate()-1 ,20) and record_time < convert(varchar(10),getdate() ,20)  
              and server_id > 0
          group by datepart(yy,record_time) ,datepart(m,record_time),datepart(d,record_time), 
                         datepart(hh,record_time), datepart(n,record_time)) w
group by w.year,  w.month, w.day

select top 1 @str1=convert(char(4), w.year) + '-' + convert(char(2), w.month) + '-' + convert(char(2), w.day) + ' ' + convert(char(2), w.hour) + ':' + convert(char(2), w.min)
from (select datepart(yy,record_time) Year ,datepart(m,record_time) Month,datepart(d,record_time) Day, 
                    datepart(hh,record_time) Hour, datepart(n,record_time) Min, sum(world_user) max_user
          from user_count with (nolock)
          where record_time >= convert(varchar(10),getdate()-1 ,20) and record_time < convert(varchar(10),getdate() ,20)  
              and server_id > 0
          group by datepart(yy,record_time) ,datepart(m,record_time),datepart(d,record_time), 
                         datepart(hh,record_time), datepart(n,record_time)) w
group by w.year,  w.month, w.day, w.hour, w.min
having max(w.max_user) = @act3
order by w.year,  w.month, w.day, w.hour, w.min

-- Daily Active user count--
select @act4=isnull(count(account), 0)
from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-1 ,20) and last_login < convert(varchar(10),getdate() ,20)

-- Total game Avg connected account --
select @act5=isnull(avg(w.avg_user), 0)
from (select datepart(yy,record_time) Year ,datepart(m,record_time) Month,datepart(d,record_time) Day, 
                    datepart(hh,record_time) Hour, datepart(n,record_time) Min, sum(world_user) avg_user
          from user_count with (nolock)
          where record_time >= convert(varchar(10),getdate()-1 ,20) and record_time < convert(varchar(10),getdate() ,20)  
              and server_id > 0
          group by datepart(yy,record_time) ,datepart(m,record_time),datepart(d,record_time), 
                         datepart(hh,record_time), datepart(n,record_time)) w
group by w.year,  w.month, w.day

-- max user per world--
select @act9=isnull(max(world_user), 0)
from user_count
where  record_time between convert(varchar(10),getdate()-1 ,20) and convert(varchar(10),getdate() ,20)
   and  server_id = 1

select top 1 @str3 = convert(char(16), record_time, 20)
from user_count
where  record_time between convert(varchar(10),getdate()-1 ,20) and convert(varchar(10),getdate() ,20)
   and  server_id = 1
group by record_time, world_user
having world_user = @act9

select @act10=isnull(max(world_user), 0)
from user_count
where  record_time between convert(varchar(10),getdate()-1 ,20) and convert(varchar(10),getdate() ,20)
   and  server_id = 2

select top 1 @str4 = convert(char(16), record_time, 20)
from user_count
where  record_time between convert(varchar(10),getdate()-1 ,20) and convert(varchar(10),getdate() ,20)
   and  server_id = 2
group by record_time, world_user
having world_user = @act10

-- Total game Min connected account --
select @act11=isnull(min(w.min_user), 0)
from (select datepart(yy,record_time) Year ,datepart(m,record_time) Month,datepart(d,record_time) Day, 
                    datepart(hh,record_time) Hour, datepart(n,record_time) Min, sum(world_user) min_user
          from user_count with (nolock)
          where record_time >= convert(varchar(10),getdate()-1 ,20) and record_time < convert(varchar(10),getdate() ,20)  
              and server_id > 0
          group by datepart(yy,record_time) ,datepart(m,record_time),datepart(d,record_time), 
                         datepart(hh,record_time), datepart(n,record_time)) w
group by w.year,  w.month, w.day

select top 1 @str2=convert(char(4), w.year) + '-' + convert(char(2), w.month) + '-' + convert(char(2), w.day) + ' ' + convert(char(2), w.hour) + ':' + convert(char(2), w.min)
from (select datepart(yy,record_time) Year ,datepart(m,record_time) Month,datepart(d,record_time) Day, 
                    datepart(hh,record_time) Hour, datepart(n,record_time) Min, sum(world_user) min_user
          from user_count with (nolock)
          where record_time >= convert(varchar(10),getdate()-1 ,20) and record_time < convert(varchar(10),getdate() ,20)  
              and server_id > 0
          group by datepart(yy,record_time) ,datepart(m,record_time),datepart(d,record_time), 
                         datepart(hh,record_time), datepart(n,record_time)) w
group by w.year,  w.month, w.day, w.hour, w.min
having max(w.min_user) = @act11
order by w.year,  w.month, w.day, w.hour, w.min

insert st_acc_daily(date, Total_acc, New_acc, Max_con, Max_con_time, Active_acc, Avg_con, Max01, Max01_time, Max02, Max02_time, Min_con, Min_con_time, Login_count, Account_count, Avg_playing_time, Total_playing_time)
values ( convert(varchar(10),getdate()-1,111),@act1,@act2,@act3,@str1,@act4,@act5,@act9,@str3,@act10,@str4,@act11,@str2, @act12, @act13, @act14, @act15)

end
GO
/****** Object:  StoredProcedure [dbo].[Pr_acc_act]    Script Date: 01/22/2023 15:56:51 ******/
SET ANSI_NULLS ON
GO
SET QUOTED_IDENTIFIER ON
GO
--exec Pr_acc_act
--select * from st_acc_active

create    procedure [dbo].[Pr_acc_act]
as
begin

declare @act1 int
declare @act2 int
declare @act3 int
declare @act4 int
declare @act5 int
declare @act6 int
declare @act7 int
declare @act8 int
declare @act9 int
declare @act10 int

select @act1=count(*) from user_account with (nolock)

select @act2=count(*) from user_account with (nolock)
where last_login is null

select @act3=count(*) from user_account with (nolock)
where last_login is not null

select @act4=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-90 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act5=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-60 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act6=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-30 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act7=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-15 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act8=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-7 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act9=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-3 ,20) and last_login < convert(varchar(10),getdate() ,20)

select @act10=count(*) from user_account with (nolock)
where last_login >=  convert(varchar(10),getdate()-1 ,20) and last_login < convert(varchar(10),getdate() ,20)

insert st_acc_active values ( convert(varchar(8),getdate()-1,112),@act1,@act2,@act3,@act4,@act5,@act6,@act7,@act8,@act9,@act10)

end
GO
/****** Object:  Default [DF_user_info_create_date]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_info] ADD  CONSTRAINT [DF_user_info_create_date]  DEFAULT (getdate()) FOR [create_date]
GO
/****** Object:  Default [DF_user_info_status_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_info] ADD  CONSTRAINT [DF_user_info_status_flag]  DEFAULT ((0)) FOR [status_flag]
GO
/****** Object:  Default [DF_user_info_kind]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_info] ADD  CONSTRAINT [DF_user_info_kind]  DEFAULT ((0)) FOR [kind]
GO
/****** Object:  Default [DF_user_info_userkind]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_info] ADD  CONSTRAINT [DF_user_info_userkind]  DEFAULT ((0)) FOR [userkind]
GO
/****** Object:  Default [DF_user_info_service_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_info] ADD  CONSTRAINT [DF_user_info_service_flag]  DEFAULT ((0)) FOR [service_flag]
GO
/****** Object:  Default [DF_user_data_create_date]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_data] ADD  CONSTRAINT [DF_user_data_create_date]  DEFAULT (getdate()) FOR [create_date]
GO
/****** Object:  Default [DF__user_data__subjo__38996AB5]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_data] ADD  DEFAULT ((-1)) FOR [subjob0_class]
GO
/****** Object:  Default [DF__user_data__subjo__398D8EEE]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_data] ADD  DEFAULT ((-1)) FOR [subjob1_class]
GO
/****** Object:  Default [DF__user_data__subjo__3A81B327]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_data] ADD  DEFAULT ((-1)) FOR [subjob2_class]
GO
/****** Object:  Default [DF__user_data__subjo__3B75D760]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_data] ADD  DEFAULT ((-1)) FOR [subjob3_class]
GO
/****** Object:  Default [DF_user_count_record_time]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_count] ADD  CONSTRAINT [DF_user_count_record_time]  DEFAULT (getdate()) FOR [record_time]
GO
/****** Object:  Default [DF_user_count_dayofweek]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_count] ADD  CONSTRAINT [DF_user_count_dayofweek]  DEFAULT (datepart(weekday,getdate())) FOR [dayofweek]
GO
/****** Object:  Default [DF_user_auth_new_pwd_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_auth] ADD  CONSTRAINT [DF_user_auth_new_pwd_flag]  DEFAULT ((0)) FOR [new_pwd_flag]
GO
/****** Object:  Default [DF_user_account__pay_stat]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account__pay_stat]  DEFAULT ((0)) FOR [pay_stat]
GO
/****** Object:  Default [DF_user_account__login_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account__login_flag]  DEFAULT ((0)) FOR [login_flag]
GO
/****** Object:  Default [DF_user_account__warn_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account__warn_flag]  DEFAULT ((0)) FOR [warn_flag]
GO
/****** Object:  Default [DF_user_account__block_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account__block_flag]  DEFAULT ((0)) FOR [block_flag]
GO
/****** Object:  Default [DF_user_account__block_flag2]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account__block_flag2]  DEFAULT ((0)) FOR [block_flag2]
GO
/****** Object:  Default [DF_user_account_subscription_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account_subscription_flag]  DEFAULT ((0)) FOR [subscription_flag]
GO
/****** Object:  Default [DF_user_account_forbidden_servers]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account_forbidden_servers]  DEFAULT ((0)) FOR [forbidden_servers]
GO
/****** Object:  Default [DF_user_account_premiunm_expire]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  CONSTRAINT [DF_user_account_premiunm_expire]  DEFAULT ((0)) FOR [premium_expire]
GO
/****** Object:  Default [DF__user_acco__cheat__31EC6D26]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  DEFAULT ((0)) FOR [cheat]
GO
/****** Object:  Default [DF__user_acco__speci__32E0915F]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  DEFAULT ((0)) FOR [special_gates]
GO
/****** Object:  Default [DF__user_acco__donat__33D4B598]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[user_account] ADD  DEFAULT ((0)) FOR [donate]
GO
/****** Object:  Default [DF_ssn_newsletter]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[ssn] ADD  CONSTRAINT [DF_ssn_newsletter]  DEFAULT ((0)) FOR [newsletter]
GO
/****** Object:  Default [DF_ssn_reg_date]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[ssn] ADD  CONSTRAINT [DF_ssn_reg_date]  DEFAULT (getdate()) FOR [reg_date]
GO
/****** Object:  Default [DF_ssn_account_num]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[ssn] ADD  CONSTRAINT [DF_ssn_account_num]  DEFAULT ((0)) FOR [account_num]
GO
/****** Object:  Default [DF_ssn_status_flag]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[ssn] ADD  CONSTRAINT [DF_ssn_status_flag]  DEFAULT ((0)) FOR [status_flag]
GO
/****** Object:  Default [DF_ssn_final_master_date]    Script Date: 01/22/2023 15:56:49 ******/
ALTER TABLE [dbo].[ssn] ADD  CONSTRAINT [DF_ssn_final_master_date]  DEFAULT (getdate()) FOR [final_master_date]
GO
/****** Object:  Default [DF_banancheg_uid]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[banancheg] ADD  CONSTRAINT [DF_banancheg_uid]  DEFAULT ((0)) FOR [uid]
GO
/****** Object:  Default [DF_banancheg_cid]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[banancheg] ADD  CONSTRAINT [DF_banancheg_cid]  DEFAULT ((0)) FOR [cid]
GO
/****** Object:  Default [DF_banancheg_bid]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[banancheg] ADD  CONSTRAINT [DF_banancheg_bid]  DEFAULT ((0)) FOR [bid]
GO
/****** Object:  Default [DF_banancheg_kicked]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[banancheg] ADD  CONSTRAINT [DF_banancheg_kicked]  DEFAULT ((1)) FOR [kicked]
GO
/****** Object:  Default [DF_gm_illegal_login_try_date]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[gm_illegal_login] ADD  CONSTRAINT [DF_gm_illegal_login_try_date]  DEFAULT (getdate()) FOR [try_date]
GO
/****** Object:  Default [DF__worldstat__statu__5CD6CB2B]    Script Date: 01/22/2023 15:56:51 ******/
ALTER TABLE [dbo].[worldstatus] ADD  CONSTRAINT [DF__worldstat__statu__5CD6CB2B]  DEFAULT ((0)) FOR [status]
GO
