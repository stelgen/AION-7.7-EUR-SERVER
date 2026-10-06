#ifndef __SECURITY_H__
#define __SECURITY_H__

#include "../catx/catxstring.h"
#include <map>
#include <vector>
#include <QMutex>

#include <time.h>

using namespace std;
using namespace CatX;

enum
{
	E_SECURITY_DDOS = 0,
	E_SECURITY_BRUT = 1,
	E_SECURITY_INVALID_PACKET = 2,
	E_SECURITY_INVALID_PASSWORD = 3
};

class Security
{
	private:
		enum
		{
			E_BANNED = 0,
			E_OK = 1
		};
		enum
		{
			E_BAN_DDOS = 3600,
			E_BAN_BRUT = 5,
			E_BAN_MAX = 4000,
			E_BAN_INVALID_PACKET = 5,
			E_LOGIN_ATTEMPS = 2
		};
		int maxThreads_;
		String fileName_;
		map <String, int> blackIpData_;			// ip -> Banned or Not
		map <String, int> illegalPassword_;		// ip -> invalid passwords in row
		map <String, time_t> banTime_;			// ip -> time
		QMutex mutex_;
	public:
		Security (const int maxThreads, const String & filename);
		bool checkTime (const String & ip);
		void ban (const String & ip, const int reason, int threads = 0);
		void initTime (const String & ip);
		void loadBlackList ();
		int maxThreads ();
		bool isBlackIp (const String & ip);
		bool isDDoS (const int runningThreads);
		void incIllegal (const String & ip);
		void clearIllegal (const String & ip);
};

#endif // __SECURITY_H__
