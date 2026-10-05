# STEP 1: move zombie duplicate rows to free slots 35/36/37 (char 1002, warehouse=0)
# rows: 195 (slot0 potions 160020008), 196 (slot1 169300002), 198 (slot2 stigma 162005039)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== BEFORE: full snapshot of moved rows =='
Q "SELECT id, slot_id, name_id, amount, tid, CONVERT(varchar,update_date,120) FROM [_AionWorldNew114_rc].dbo.user_item WHERE id IN (195,196,198)"
$c=$con.CreateCommand(); $c.CommandTimeout=60
$c.CommandText="UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=35 WHERE id=195; UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=36 WHERE id=196; UPDATE [_AionWorldNew114_rc].dbo.user_item SET slot_id=37 WHERE id=198;"
'rows updated: ' + $c.ExecuteNonQuery()
'== AFTER: slots 0-2 now single =='
Q "SELECT slot_id, name_id, amount FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id IN (0,1,2) ORDER BY slot_id"
'== AFTER: moved rows =='
Q "SELECT id, slot_id, name_id, amount FROM [_AionWorldNew114_rc].dbo.user_item WHERE id IN (195,196,198)"
'== dup check =='
Q "SELECT slot_id, COUNT(*) FROM [_AionWorldNew114_rc].dbo.user_item WHERE char_id=1002 AND warehouse=0 AND slot_id>=0 AND slot_id<=40 GROUP BY slot_id HAVING COUNT(*)>1"
'BACKUP NOTE: 195 was slot_id=0, 196 slot_id=1, 198 slot_id=2 (all warehouse=0, update_date unchanged on purpose? NOTE: UPDATE may bump nothing - update_date is manual)'
