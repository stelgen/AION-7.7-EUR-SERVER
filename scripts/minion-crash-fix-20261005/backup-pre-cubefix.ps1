# BACKUP: full DB + row-level (before cube fix)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
$c=$con.CreateCommand(); $c.CommandTimeout=300
$c.CommandText="BACKUP DATABASE [_AionWorldNew114_rc] TO DISK='D:\_REF58\prod-backups\AionWorld-20261005-pre-cubefix.bak' WITH COMPRESSION, INIT"
$sw=[Diagnostics.Stopwatch]::StartNew()
$c.ExecuteNonQuery() | Out-Null
$sw.Stop()
'FULL BACKUP OK: ' + $sw.Elapsed.TotalSeconds + 's'
$c2=$con.CreateCommand(); $c2.CommandTimeout=120
$c2.CommandText="IF OBJECT_ID('_AionWorldNew114_rc.dbo.z_backup_useritem_cubefix') IS NOT NULL DROP TABLE [_AionWorldNew114_rc].dbo.z_backup_useritem_cubefix; SELECT * INTO [_AionWorldNew114_rc].dbo.z_backup_useritem_cubefix FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0"
'ROW BACKUP TABLE rows: ' + $c2.ExecuteNonQuery()
$c3=$con.CreateCommand(); $c3.CommandTimeout=60
$c3.CommandText="SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.z_backup_useritem_cubefix"
'row backup count: ' + $c3.ExecuteScalar()
