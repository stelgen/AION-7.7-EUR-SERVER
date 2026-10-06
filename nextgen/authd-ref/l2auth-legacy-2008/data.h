#ifndef __DATA_S__
#define __DATA_S__

#define ERROR_FILE_NAME "error.log"

#include "catx/parser.h"
#include "catx/catxstring.h"
#include "serverlist.h"
#include "serverclasses/iddatamap.h"

enum
{
	REVISION_785a = 0,
	REVISION_c621 = 1	
};

using namespace CatX;

namespace _Data
{
	class SqlData
	{
		private:
			String sqlServer_;
			String sqlDb_;
			String sqlServerTable_;
		public:
			SqlData () : sqlServerTable_ ("server") {}
			void setServer (const String & ser) {sqlServer_ = ser;}
			String getServer () const {return sqlServer_;}
			void setDb (const String & db) {sqlDb_ = db;}
			String getDb () const {return sqlDb_;}
			void setTable (const String & table) {sqlServerTable_ = table;}
			String getTable () const {return sqlServerTable_;}
	};

	class ServerData
	{
		private:
			int revision_;
			int intPort_;
			int exPort_;
			int maxThreads_;
			String blowfish_;
			bool autoReg_;
			int maxErrorLevel_;
			int gameGuard_;
		public:
			ServerData () : maxThreads_ (200), autoReg_ (false), maxErrorLevel_ (0), gameGuard_ (0) {}
			int getRevision () const {return revision_;}
			void setRevision (int rev) {revision_ = rev;}
			int getIntPort () const {return intPort_;}
			void setIntPort (int port) {intPort_ = port;}
			int getExPort () const {return exPort_;}
			void setExPort (int port) {exPort_ = port;}
			int getMaxThreads () const {return maxThreads_;}
			void setMaxThreads (int am) {maxThreads_ = am;}
			String getBlowfish () const {return blowfish_;}
			void setBlowfish (const String & bf) {blowfish_ = bf;}			
			bool getAutoReg () const {return autoReg_;}
			void setAutoReg (bool val) {autoReg_ = val;}
			int getMaxErrorLevel () const {return maxErrorLevel_;}
			void setMaxErrorLevel (int er) {maxErrorLevel_ = er;}
			int getGameGuard () const {return gameGuard_;}
			void setGameGuard (int gg) {gameGuard_ = gg;}
	};
	
	class LogData
	{
		private:
			bool log_;
			String db_;
			String table_;
		public:
			LogData () : log_ (false) {}
			void setOn (bool val) {log_ = val;}
			bool isOn () const {return log_;}
			void setDb (const String & db) {db_ = db;}
			String getDb () const {return db_;}
			void setTable (const String & table) {table_ = table;}
			String getTable () const {return table_;}
	};

	class IpLockData
	{
		private:
			bool lock_;
			String db_;
			String tableIp_;
			String tableControl_;
		public:
			IpLockData () : lock_ (false) {}
			void setOn (bool val) {lock_ = val;}
			bool isOn () const {return lock_;}
			void setDb (const String & db) {db_ = db;}
			String getDb () const {return db_;}
			void setTableIp (const String & db) {tableIp_ = db;}
			String getTableIp () const {return tableIp_;}
			void setTableControl (const String & db) {tableControl_ = db;}
			String getTableControl () const {return tableControl_;}
	};

	class OnlineData
	{
		private:
			bool online_;
			String db_;
			String table_;
			int time_;
		public:
			OnlineData () : online_ (false), time_ (60) {}
			void setOn (bool val) {online_ = val;}
			bool isOn () const {return online_;}
			String getDb () const {return db_;}
			void setDb (const String & db) {db_ = db;}
			String getTable () const {return table_;}
			void setTable (const String & table) {table_ = table;}
			int getTimeUpdate () const {return time_;}
			void setTimeUpdate (int time) {time_ = time;}
	};
}




class Data
{
	private:
		static ServerVector serverVector_;
		static IdDataMap idDataMap_;
		static _Data::SqlData sql_;
		static _Data::ServerData server_;
		static _Data::LogData log_;
		static _Data::IpLockData ip_;
		static _Data::OnlineData online_;
		static map <int, pair <bool, int> > playerOnline_;
	public:
		static _Data::SqlData sql () {return sql_;}
		static _Data::ServerData server () {return server_;}
		static _Data::LogData log () {return log_;}
		static _Data::IpLockData ip () {return ip_;}
		static _Data::OnlineData online () {return online_;}
		static ServerVector & gServerList () {return serverVector_;}
		static IdData & idData (int id) {return idDataMap_ [id];}
		static IdDataMap & idDataMap () {return idDataMap_;}
		static void initData (Parser * myP);
		static void setOnline (int uid, bool val, int server);
		static bool getOnline (int uid);
		
};

class Error
{
	public:
		static void writeError (const String & str);
};

#endif
