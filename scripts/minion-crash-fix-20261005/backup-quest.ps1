# READ-ONLY: backup user_quest + quest XML classification
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
$c=$con.CreateCommand(); $c.CommandTimeout=120
$c.CommandText="IF OBJECT_ID('_AionWorldNew114_rc.dbo.z_backup_userquest_cubefix') IS NOT NULL DROP TABLE [_AionWorldNew114_rc].dbo.z_backup_userquest_cubefix; SELECT * INTO [_AionWorldNew114_rc].dbo.z_backup_userquest_cubefix FROM [_AionWorldNew114_rc].dbo.user_quest WHERE char_id=1002"
'quest backup rows: ' + $c.ExecuteNonQuery()
