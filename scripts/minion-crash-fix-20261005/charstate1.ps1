# READ-ONLY discovery: find char data tables and columns (no writes)
$con = New-Object System.Data.SqlClient.SqlConnection("Server=.;Integrated Security=True")
$con.Open()
function Q($q) {
  $c=$con.CreateCommand(); $c.CommandTimeout=60; $c.CommandText=$q
  $r=$c.ExecuteReader(); $out=@()
  while($r.Read()){ $o=@(); for($i=0;$i -lt $r.FieldCount;$i++){ $o += [string]$r.GetValue($i) }; $out += ($o -join ' | ') }
  $r.Close(); $out
}
'== DATABASES =='
Q "SELECT name FROM sys.databases ORDER BY name"
'== tables user%/char% across dbs =='
$dbs = Q "SELECT name FROM sys.databases WHERE name NOT IN ('master','tempdb','model','msdb') ORDER BY name"
foreach ($db in $dbs) {
  $t = Q "SELECT TABLE_NAME FROM [$db].INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME LIKE 'user%' OR TABLE_NAME LIKE 'char%' ORDER BY TABLE_NAME"
  if ($t) { "=== $db ==="; $t }
}
