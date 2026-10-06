#ifndef __PACKETS_H__
#define __PACKETS_H__

#include "catx/catxstring.h"
#include "serverclasses/bfish.h"
#include <vector>
#include <map>
#include <iostream>

#include <QByteArray>

#include "data.h"
#include "b32.h"

using namespace std;

enum
{
	E_CS_REQUEST_AUTH_LOGIN_C4 = 40,
	E_CS_REQUEST_AUTH_LOGIN_CT1 = 160,
	E_CS_REQUEST_SERVER_LOGIN_CT1 = 16,
	E_CS_REQUEST_SERVER_LIST_CT1 = 16
};

enum
{
	E_PCK_REQUEST_AUTH_LOGIN = 0,
	E_PCK_REQUEST_SERVER_LOGIN = 0x02,
	E_PCK_REQUEST_SERVER_LIST = 0x05,
	E_PCK_EULA = 0x07
};

class Packet
{
	private:
		static String checksum_ (const String &);
		static String ip_ (const String &);
		static String dword_ (const int);
		static String scrambleRSA_ (const String &);
	public:
		static String init (const String & key, const String & bfish);
		static String initC4 ();
		static String serverList (int lastworld);
		static String authok (const String &);
		static String authfail (const int);
		static String playOk (const String &);
		static String playFail ();
		static String eula ();
		static bool checkPckSum (const Blowfish &, const String & pck);
		static int intFromString (const String & str, int pos);
};

class GamePacket 
{
	public:
		static String ping ();
		static String playerLogin (const String &, int);
		static String package (const String &);
		static String authKick (int uid, const String & account);
};

#endif // __PACKETS_H__
