# READ-ONLY: orphan sub-rows check (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== user_item_option schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_item_option' ORDER BY ORDINAL_POSITION"
'== user_item_enslave_stone schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_item_enslave_stone' ORDER BY ORDINAL_POSITION"
'== duplicate slot check (explicit) =='
Q "SELECT slot, slot_id, COUNT(*) AS cnt FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id>=0 AND slot_id<=40 GROUP BY slot, slot_id HAVING COUNT(*)>1 ORDER BY slot, slot_id"
'== slots near zero with all rows =='
Q "SELECT slot, slot_id, name_id, amount FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id IN (0,1,2) ORDER BY slot_id, update_date"
