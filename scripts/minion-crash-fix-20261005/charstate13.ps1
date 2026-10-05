# READ-ONLY: duplicate slots + orphans (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== DUP SLOTS (cube) =='
Q "SELECT slot, slot_id, COUNT(*) AS cnt FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id>=0 AND slot_id<=40 GROUP BY slot, slot_id HAVING COUNT(*)>1 ORDER BY slot, slot_id"
'== slots 0-2 detail =='
Q "SELECT slot, slot_id, name_id, amount FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id IN (0,1,2) ORDER BY slot_id, update_date"
'== enslave_stone schema =='
Q "SELECT COLUMN_NAME, DATA_TYPE FROM [_AionWorldNew114_rc].INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME='user_item_enslave_stone' ORDER BY ORDINAL_POSITION"
'== item_option orphan count (option rows without item) =='
Q "SELECT COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_item_option o WHERE o.char_id=1002 AND NOT EXISTS (SELECT 1 FROM [_AionWorldNew114_rc].dbo.user_item i WHERE i.id=o.id)"
'== option rows for char =='
Q "SELECT TOP 5 o.id, o.soul_bound, o.enchant_count FROM [_AionWorldNew114_rc].dbo.user_item_option o WHERE o.char_id=1002"
