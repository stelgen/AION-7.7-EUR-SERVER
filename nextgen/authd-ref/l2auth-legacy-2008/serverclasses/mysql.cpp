#include "mysql.h"

void DbConnect::init (const String & server, const String & lin2db)
{
	db_ = QSqlDatabase::addDatabase ("QODBC");
	db_.setDatabaseName("Driver={SQL Native Client};Server=" + QString (server.c_str ()) + ";Database=" + QString (lin2db.c_str ()) + ";Trusted_Connection=yes;");
}

void DbConnect::sql_connect ()
{
	mutex_.lock ();
	db_.setConnectOptions ("SQL_ATTR_LOGIN_TIMEOUT=5");
	while (1)
	{
		if (db_.open ())
		{
			break;
		}
		else
		{
			db_.setConnectOptions ("SQL_ATTR_LOGIN_TIMEOUT=5");
		}
	}
}

void DbConnect::sql_close ()
{
	db_.close ();
	mutex_.unlock ();
}



MySQL::MySQL ()
{
	String server = Data::sql ().getServer ();
	String lin2db = Data::sql ().getDb ();
	dbConnect_.init (server, lin2db);
}

void MySQL::run ()
{
}

String MySQL::dbCrypt_ (const String & str)
{
	char * key = (char *) calloc (17, sizeof (char));
	char * dst = (char *) calloc (17, sizeof (char));
	int i = 0;
	int nBytes = str.length ();
	if (nBytes > 16 || nBytes < 4)
	{
		return "";
	}
	long long rslt, one, two, three, four;
	while (i < nBytes)
	{
		++i;
		key[i] = str[i - 1];
		dst[i] = key[i];
	}
	rslt = (unsigned int)(key[1] + (key[2] << 8) + (key[3] << 16) + (key[4] << 24));
	one = (unsigned int) (rslt * 213119 + 2529077); 
	rslt = (unsigned int) (key[5] + (key[6] << 8) + (key[7] << 16) + (key[8] << 24));
	two = (unsigned int) (rslt * 213247 + 2529089);
	rslt = (unsigned int) (key[9] + (key[10] << 8) + (key[11] << 16) + (key[12] << 24));
	three = (unsigned int)(rslt * 213203 + 2529589);
	rslt = (unsigned int)(key[13] + (key[14] << 8) + (key[15] << 16) + (key[16] << 24));
	four = (unsigned int) (rslt * 213821 + 2529997);
	key[4] = one >> 24;
	key[3] = (one - (key[4] << 24)) >> 16;
	key[2] = (one - (key[4] << 24) - (key[3] << 16)) >> 8;
	key[1] = one - (key[4] << 24) - (key[3] << 16) - (key[2] << 8);
	key[8] = two >> 24;     
	key[7] = (two - (key[8] << 24)) >> 16;
	key[6] = (two - (key[8] << 24) - (key[7] << 16)) >> 8;
	key[5] = two - (key[8] << 24) - (key[7] << 16) - (key[6] << 8);
	key[12] = three >> 24;     
	key[11] = (three - (key[12] << 24)) >> 16;
	key[10] = (three - (key[12] << 24) - (key[11] << 16)) >> 8;
	key[9] = three - (key[12] << 24) - (key[11] << 16) - (key[10] << 8);
	key[16] = four >> 24;     
	key[15] = (four - (key[16] << 24)) >> 16;
	key[14] = (four - (key[16] << 24) - (key[15] << 16)) >> 8;
	key[13] = four - (key[16] << 24) - (key[15] << 16) - (key[14] << 8);
	dst[1] = dst[1] ^ key[1];
	i = 1;
	while (i++ < 16)
	{
		dst[i] = dst[i] ^ dst[i-1] ^ key[i];
	}
	i = 0;
	while (i++ < 16)
	{
		if (dst[i] == 0) 
		{
			dst[i] = 102;
		}
	}
	i = 0;
	String encrypt;
	while (i++ < 16)
	{
		encrypt += dst[i];
	}
	free (key);
	free (dst);
	return encrypt;
}

void MySQL::slotCheckLogin (String acc, int id, String pas, String ip)
{
	String account;
	String pass;
	for (unsigned int i = 0; i < acc.length (); ++i)
	{
		if (acc[i] == (char) 0)
		{
			break;
		}
		account += acc[i];
	}
	for (unsigned int i = 0; i < pas.length (); ++i)
	{
		if (pas[i] == (char) 0)
		{
			break;
		}
		pass += pas[i];
	}
	for (unsigned int i = 0; i < account.length (); ++i)
	{
		if (!(
			(account[i] >= 'a' && account[i] <= 'z') ||
			(account[i] >= 'A' && account[i] <= 'Z') ||
			(account[i] >= '0' && account[i] <= '9')
		))
		{
			emit signalCheckLogin (acc, id, 0, 0, SYSTEM_ERROR);
			return;
		}
	}
	for (unsigned int i = 0; i < pass.length (); ++i)
	{
		if (!(
			(pass[i] >= 'a' && pass[i] <= 'z') ||
			(pass[i] >= 'A' && pass[i] <= 'Z') ||
			(pass[i] >= '0' && pass[i] <= '9')
		))
		{
			emit signalCheckLogin (acc, id, 0, 0, SYSTEM_ERROR);
			return;
		}
	}	
	if (!pass.length ())
	{
		emit signalCheckLogin (acc, id, 0, 0, SYSTEM_ERROR);
		return;
	}
	dbConnect_.sql_connect ();
	QSqlQuery query ("SELECT uid, pay_stat, block_flag, block_flag2, last_world FROM "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_account WHERE account = '" + QString (account.c_str ()) +"'");
	int lastWorldNo = query.record ().indexOf ("last_world");
	int uidNo = query.record ().indexOf ("uid");
	int psNo = query.record ().indexOf ("pay_stat");
	int b1No = query.record ().indexOf ("block_flag");
	int b2No = query.record ().indexOf ("block_flag2");
	bool selected = false;                                
	int uid = 0;	
	int last_world = 0;
	while (query.next ())
	{
		selected = true;
		uid = query.value (uidNo).toInt ();
		last_world = query.value (lastWorldNo).toInt ();
		if (!query.value (psNo).toBool ())
		{
			dbConnect_.sql_close ();
			emit signalCheckLogin (acc, id, 0, 0, ACCESS_FORBIDDEN);
			return;
		}
		if (query.value (b2No).toBool () || query.value (b1No).toBool ())
		{
			dbConnect_.sql_close ();
			emit signalCheckLogin (acc, id, 0, 0, ACCOUNT_BANNED);
			return;
		}
		if (Data::ip ().isOn ())
		{
			if (!checkIp_ (account, ip))
			{
				dbConnect_.sql_close ();
				emit signalCheckLogin (acc, id, 0, 0, WRONG_IP);
				return;
			}
		}
		break;
	}
	if (!selected)
	{
		if (Data::server ().getAutoReg ())
		{
			dbConnect_.sql_close ();
			int retuid = register_ (account, pass);
			emit signalCheckLogin (acc, id, retuid, 0, LOGIN_OK);
			return;
		}
		else
		{
			dbConnect_.sql_close ();
			emit signalCheckLogin (acc, id, 0, 0, INVALID_LOGIN);
			return;
		}
	}
	if (check_pwd_ (account, pass))
	{
		dbConnect_.sql_close ();
		if (!Data::getOnline (uid))
		{
			emit signalCheckLogin (acc, id, uid, last_world, LOGIN_OK);
		}
		else
		{
			emit signalCheckLogin (acc, id, uid, last_world, ACCOUNT_IN_USE);
		}
		return;
	}
	else
	{
		dbConnect_.sql_close ();
		emit signalCheckLogin (acc, id, 0, 0, INVALID_PASSWORD);
		return;
	}
}

void MySQL::slotUpdateOnline (String field, int online)
{
	dbConnect_.sql_connect ();
	String db = Data::online ().getDb ();
	QSqlQuery query ("SELECT * FROM "+ QString (db.c_str ()) +".dbo." + QString (Data::online ().getTable ().c_str ()) + " WHERE ip = '" + QString (field.c_str ()) + "'");
	while (query.next ())
	{
		QSqlQuery quer ("UPDATE "+ QString (db.c_str ()) +".dbo." + QString (Data::online ().getTable ().c_str ()) + " SET online = " + String (online).c_str () + "WHERE ip = '" + QString (field.c_str ()) + "'");
		quer.next ();
		dbConnect_.sql_close ();
		return;
	}
	QSqlQuery quer ("INSERT INTO "+ QString (db.c_str ()) +".dbo." + QString (Data::online ().getTable ().c_str ()) + " VALUES ('" + QString (field.c_str ()) + "', '" + String (online).c_str () + "')");
	quer.next ();
	dbConnect_.sql_close ();
}	
	
void MySQL::slotUserLogin (int id, String ip, int server_id)
{
	dbConnect_.sql_connect ();
	QTime tm;
	QDate dt;
	String date = QDate ().currentDate ().month ();
	date += '/';
	date.append (QDate ().currentDate ().day ()); 
	date += '/';
	date.append (QDate ().currentDate ().year ());
	date += ' ';
	date += tm.currentTime ().toString ().toStdString ();
	QSqlQuery quer ("UPDATE "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_account SET last_login = '" + 
									QString (date.c_str ()) + 
									"' WHERE uid = " + 
									QString (String (id).c_str ()));
	quer.next ();
	QSqlQuery que ("UPDATE "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_account SET last_world = " + 
									QString (String (server_id).c_str ()) + 
									" WHERE uid = " + 
									QString (String (id).c_str ()));
	que.next ();
	QSqlQuery query ("UPDATE "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_account SET last_ip = '" + 
						QString (ip.c_str ()) +
						"' WHERE uid = " +
						QString (String (id).c_str ()));
	query.next ();
	dbConnect_.sql_close ();
}

void MySQL::slotUserLogout (int id)
{
	dbConnect_.sql_connect ();
	QTime tm;
	QDate dt;
	String date = QDate ().currentDate ().month ();
	date += '/';
	date.append (QDate ().currentDate ().day ()); 
	date += '/';
	date.append (QDate ().currentDate ().year ());
	date += ' ';
	date += tm.currentTime ().toString ().toStdString ();
	QSqlQuery quer ("UPDATE "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_account SET last_logout = '" + 
									QString (date.c_str ()) + 
									"' WHERE uid = " + 
									QString (String (id).c_str ()));
	quer.next ();	
	dbConnect_.sql_close ();
}

void MySQL::slotLogLogin (String login, String ip, String serverIp)
{
	dbConnect_.sql_connect ();
	QTime tm;
	QDate dt;
	String date = QDate ().currentDate ().month ();
	date += '/';
	date.append (QDate ().currentDate ().day ()); 
	date += '/';
	date.append (QDate ().currentDate ().year ());
	date += ' ';
	date += tm.currentTime ().toString ().toStdString ();
	QSqlQuery quer ("INSERT INTO "+ QString (Data::log ().getDb ().c_str ()) +".dbo." + QString (Data::log ().getTable ().c_str ()) + " VALUES ('" + 
			QString (date.c_str ()) + "', '" +
			QString (login.c_str ()) + "', '" +
			QString (ip.c_str ()) + "', '" +
			QString (serverIp.c_str ()) + "')");
	quer.next ();	
	dbConnect_.sql_close ();
}

bool MySQL::checkIp_ (const String & account, const String & ip)
{
	String db = Data::ip ().getDb ();
	QSqlQuery query ("SELECT ip_lock FROM "+ QString (db.c_str ()) +".dbo."+ QString (Data::ip().getTableControl ().c_str ()) +" WHERE account = '" +
				QString (account.c_str ()) + "'");
	int icNo = query.record ().indexOf ("ip_lock");
	while (query.next ())
	{
		if (query.value (icNo).toInt ())
		{
			QSqlQuery quer ("SELECT ip FROM " + QString (db.c_str ()) + ".dbo." + QString (Data::ip ().getTableIp ().c_str ()) + " WHERE account = '" +
				QString (account.c_str ()) + "'");
			int ipNo = quer.record ().indexOf ("ip");
			while (quer.next ())
			{
				if (ip.substr (0, quer.value (ipNo).toString ().toStdString ().length ()) == quer.value (ipNo).toString ().toStdString ())
				{
					return true;
				}
			}
			return false;
		}
		else
		{
			return true;
		}
	}
	return true;
}

int MySQL::register_ (const String & login, const String & pass)
{
	dbConnect_.sql_connect ();
	String db = Data::sql ().getDb ();
	QSqlQuery query ("INSERT INTO "+ QString (db.c_str ()) +".dbo.user_account (account,pay_stat) VALUES ('"+ QString (login.c_str ()) +"', 1)");
	query.next ();
	QSqlQuery quer ("INSERT INTO "+ QString (db.c_str ()) + ".dbo.auto_reg (account,password) VALUES ('"+ QString (login.c_str ()) + "','"+ QString (pass.c_str ()) +"')");
	quer.next ();
	QSqlQuery que ("SELECT uid FROM "+ QString (db.c_str ()) +".dbo.user_account WHERE account = '"+ QString (login.c_str ()) +"'");
	int icNo = que.record ().indexOf ("uid");
	que.next ();
	int ret = que.value (icNo).toInt ();
	dbConnect_.sql_close ();
	return ret;
}

bool MySQL::check_pwd_ (const String & login, const String & pass)
{
	if (pass.length () && login.length ())
	{
		if (!Data::server ().getAutoReg ())
		{
			QSqlQuery quer ("SELECT password FROM "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.user_auth WHERE account = '" + QString (login.c_str ()) +"'");
			int passNo = quer.record ().indexOf ("password");
			bool user_auth = false;
			while (quer.next ())
			{
				user_auth = true;         	
				String pass1 = dbCrypt_ (pass);
				String pass2 = quer.value (passNo).toString ().toStdString ();
				int errors = 0;	
				unsigned int minLen = pass1.length ();
				if (minLen > pass2.length ())
				{
					minLen = pass2.length ();
				}
				if (minLen < 10)
				{
					return false;
				}
				for (unsigned int i = 0; i < minLen; ++i)
				{
					if (pass1[i] != pass2[i])
					{
						++errors;
					}
				}
				if (errors > Data::server ().getMaxErrorLevel ())
				{
					return false;
				}
				break;
			}
			if (user_auth)
			{
				return true;
			}
			else
			{
				return false;
			}
		}
		else
		{
			QSqlQuery quer ("SELECT password FROM "+ QString (Data::sql ().getDb ().c_str ()) +".dbo.auto_reg WHERE account = '" + QString (login.c_str ()) +"'");
			int passNo = quer.record ().indexOf ("password");
			bool user_auth = false;
			while (quer.next ())
			{
				user_auth = true;
				if (pass != quer.value (passNo).toString ().toStdString ())
				{
					return false;
				}
				break;
			}
			if (user_auth)
			{
				return true;
			}
			else
			{
				return false;
			}
		}
	}
	else
	{
		return false;
	}
}

void MySQL::slotInitServerList ()
{
	dbConnect_.sql_connect ();
	QSqlQuery query ("SELECT id, ip, inner_ip, ageLimit, pk_flag, kind, port FROM " + QString (Data::sql ().getDb ().c_str ()) + ".dbo." + QString (Data::sql ().getTable ().c_str ()));
	int idNo = query.record ().indexOf ("id");
	int ipNo = query.record ().indexOf ("ip");
	int innerNo = query.record ().indexOf ("inner_ip");
	int alNo = query.record ().indexOf ("ageLimit");
	int pkNo = query.record ().indexOf ("pk_flag");
	int poNo = query.record ().indexOf ("port");
	vector <Server_s> ret;
	while (query.next ())
	{
		Server_s deb;
		deb.setId (query.value (idNo).toInt ());
		deb.setIp (query.value (ipNo).toString ().toStdString ());
		deb.setInnerIp (query.value (innerNo).toString ().toStdString ());
		deb.setPort (query.value (poNo).toInt ());
		deb.setAgeLimit (query.value (alNo).toBool ());
		deb.setPvp (query.value (pkNo).toBool ());
		deb.setOnline (0);
		deb.setMaxOnline (5000);
		deb.setIsOnline (false);
		ret.push_back (deb);
	}
	Data::gServerList ().init (ret);
	dbConnect_.sql_close ();
	emit signalInitServerListComplete ();
}


