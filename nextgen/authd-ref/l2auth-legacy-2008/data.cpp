#include "data.h"

ServerVector Data::serverVector_;
IdDataMap Data::idDataMap_;
_Data::SqlData Data::sql_;
_Data::ServerData Data::server_;
_Data::IpLockData Data::ip_;
_Data::OnlineData Data::online_;
_Data::LogData Data::log_;
map <int, int> Data::playerOnline_;

void Data::initData (Parser * myP)
{
	bool configOk = true;
	// sql
	String sql_server = myP->getString ("mssql_server");
	String sql_lin2db = myP->getString ("lin2db");
	String sql_lin2db_table = myP->getString ("lin2db_table");
	if (!sql_server.length ())
	{
		configOk = false;
		Error::writeError ("auth.ini: mssql_server is undefined");
	}
	if (!sql_lin2db.length ())
	{
		configOk = false;
		Error::writeError ("auth.ini: lin2db is undefined");
	}
	if (!sql_lin2db_table.length ())
	{
		configOk = false;
		Error::writeError ("auth.ini: lin2db_table is undefined");
	}
	sql_.setServer (sql_server);
	sql_.setDb (sql_lin2db);
	sql_.setTable (sql_lin2db_table);

	// server
	String revision = myP->getString ("revision");
	int exPort = myP->getInt ("exPort");
	int intPort = myP->getInt ("intPort");
	int maxThreads = myP->getInt ("maxThreads");
	String blowfish = myP->getString ("blowfish");
	int autoReg = myP->getInt ("auto_reg");
	int maxErrorLevel = myP->getInt ("max_error_level");
	if (!revision.length ())
	{
		configOk = false;
		Error::writeError ("auth.ini: revision is undefined");		
	}
	if (!exPort)
	{
		configOk = false;
		Error::writeError ("auth.ini: exPort is undefined");		
	}
	if (!intPort)
	{
		configOk = false;
		Error::writeError ("auth.ini: intPort is undefined");		
	}
	if (!blowfish.length ())
	{
		configOk = false;
		Error::writeError ("auth.ini: blowfish is undefined");		
	}
	if (revision == "785a")
	{
		server_.setRevision (REVISION_785a);
		server_.setBlowfish (blowfish + static_cast <char> (0));
	}
	else if (revision == "c621")
	{
		server_.setRevision (REVISION_c621);
		server_.setBlowfish (blowfish);
	}
	else
	{
		configOk = false;
		Error::writeError ("auth.ini: revision error, possible values: 785a, c621");				
	}
	server_.setIntPort (intPort);
	server_.setExPort (exPort);
	server_.setAutoReg (autoReg);
	server_.setMaxErrorLevel (maxErrorLevel);
	if (maxThreads)
	{
		server_.setMaxThreads (maxThreads);
	}
	server_.setGameGuard (static_cast <bool> (myP->getInt ("gameGuard")));

	// log
	bool logEngine = static_cast <bool> (myP->getInt ("logLoginsIntoDb"));
	String db_log = myP->getString ("db_login_log");
	String table_log = myP->getString ("table_login_log");
	log_.setOn (logEngine);
	log_.setDb (db_log);
	log_.setTable (table_log);

	// ip
	bool ipEngine = static_cast <bool> (myP->getInt ("ip_lock"));
	String db_ip = myP->getString ("db_ip_lock");
	String table_control = myP->getString ("table_ip_lock_control");
	String table_ip = myP->getString ("table_ip_lock");	
	ip_.setOn (ipEngine);
	ip_.setDb (db_ip);
	ip_.setTableControl (table_control);
	ip_.setTableIp (table_ip);

	// online
	bool onlineEngine = static_cast <bool> (myP->getInt ("online_system"));
	String db_online = myP->getString ("db_online");
	String table_online = myP->getString ("onlineTable");
	int updateTimer = myP->getInt ("updateOnline");
	online_.setOn (onlineEngine);
	online_.setDb (db_online);
	online_.setTable (table_online);
	online_.setTimeUpdate (updateTimer);
	if (!configOk)
	{
		exit (1);
	}	
}

void Error::writeError (const String & err)
{
	FILE *f = fopen (ERROR_FILE_NAME, "ab");
	fwrite (err.c_str (), sizeof (char), err.length (), f);
	fprintf (f, "\n");
	fclose (f);
}

void Data::setOnline (int uid, bool val, int server_id)
{
	if (val)
	{
		playerOnline_ [uid].first = true;
		playerOnline_ [uid].second = server_id;
	}
	else
	{
		if (playerOnline_.find (uid) != playerOnline_.end ())
		{
			playerOnline_.erase (playerOnline_.find (uid));
		}
	}
}

bool Data::getOnline (int uid)
{
	return playerOnline_[uid].first;
}

void Data::freeOnline (int server_id)
{
	for (map <int, pair <bool, int> >::const_iterator p = playerOnline_.begin (); p != playerOnline_.end (); ++p)
	{
		if (p.second == server_id)
		{
			playerOnline_.erase (p);
		}
	}
}


