#ifndef __MYSQL_H__
#define __MYSQL_H__

#include <QThread>
#include <QtSql>
#include <QMutex>
#include <iostream>
#include <vector>

#include "../catx/parser.h"
#include "../catx/catxstring.h"

#include "../data.h"

using namespace std;
using namespace CatX;

enum 
{
	LOGIN_OK = 0,
	SYSTEM_ERROR = 1,
	INVALID_PASSWORD = 2,
	INVALID_LOGIN = 3,
	ACCESS_FORBIDDEN = 4,
	ACCOUNT_IN_USE = 7,
	ACCOUNT_BANNED = 9,
	WRONG_IP = 10
};

class DbConnect
{
	private:
		QSqlDatabase db_;
		QMutex mutex_;
	public:		
		void init (const String & server, const String & db);
		void sql_connect ();
		void sql_close ();
};

class MySQL : public QThread
{
	Q_OBJECT
	private:
		bool checkIp_ (const String & account, const String & ip);
		int register_ (const String & account, const String & pwd);
		bool check_pwd_ (const String & login, const String & pass);
		String dbCrypt_ (const String &);
		DbConnect dbConnect_;
	protected:
		void run ();
	public:
		MySQL ();
	public slots:
		void slotCheckLogin (String account, int thread_id, String pass, String ip);
		void slotUpdateOnline (String, int);
		void slotUserLogin (int, String, int);
		void slotUserLogout (int);
		void slotLogLogin (String, String, String);
		void slotInitServerList ();
	signals:
		void signalInitServerListComplete ();
		void signalCheckLogin (String, int, int, int, int);
		

};

#endif // __MYSQL_H__
