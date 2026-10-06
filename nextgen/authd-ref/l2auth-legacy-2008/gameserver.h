#ifndef __GAME_SERVER__
#define __GAME_SERVER__

#include <QtNetwork>

#include "catx/catxstring.h"
#include <iostream>

#include "b32.h"
#include "data.h"
#include "packets.h"

#define PACKET_SEG 50

using namespace std;

class GameCxServer;
class GameCxInteraction;
class GameCxConnectionThread;
class GameCxConnection;

class IdDataGs : public map <int, pair <int, String> >
{
	private:
		map <int, pair <int, String> >::const_reverse_iterator getItById_ (int server_id) const;
		map <int, pair <int, String> >::const_reverse_iterator getItByIp_ (const String & ip) const;		
	public:
		int getThreadById (int server_id) const;
		int getThreadByIp (const String & ip) const;
		String getIpById (int server_id) const;
		String getIpByThread (int thread_id) const;
		int getIdByThread (int thread_id) const;
		int getIdByIp (const String & ip) const;
		void setIp (int thread_id, const String & ip) {(*this)[thread_id].second = ip;}
		void setId (int thread_id, int server_id) {(*this)[thread_id].first = server_id;}
};

enum
{
	E_GS_PCK_SESSION_KEY2 = 0,
	E_GS_ACCOUNT_IN_USE = 1,
	E_GS_PCK_PLAYER_CONNECTED = 0x02,
	E_GS_PCK_PLAYER_DISCONNECTED = 0x03,
	E_GS_PCK_ONLINE = 0x05,
	E_GS_PCK_SERVER_CONNECTED = 0x0a,
	E_GS_PCK_PACKAGE = 0x0b,
	GS_CONNECTED = 0x91,
	E_GS_PCK_SERVER_DISCONNECTED = 0xff
};


class GameCxServer :public QTcpServer
{
	Q_OBJECT
	private:
		void initPackage_ (const String & pck);
		GameCxInteraction * pInteraction_;
		QTimer * pUpdateOnlineTimer_;
		IdDataGs idDataGs_;	// thread_id -> data
	public:
		GameCxServer (QObject *pParent = 0);
		virtual ~GameCxServer(void);
	signals:
		void signalSendToThread (int thread_id, String pck);
		void signalSendSessionKey2 (String pck);
	signals:
		void signalSqlUpdateOnline (String, int);
		void signalSqlUserLogin (int, String, int);
		void signalSqlUserLogout (int);
	private slots:
		void slotAuthKick (int uid, String account);
		void slotUpdateOnline_ ();
		void slotRecvDataFromServer (int server_id, String pck);
		void slotRecvDataFromThreads (int thread_id, String pck);
	protected:
		void incomingConnection (int nSocket);
};

class GameCxInteraction: public QObject
{
	Q_OBJECT
	private:
		int id_;
		GameCxConnectionThread *CreateThread( qint32 qnSocket);
	public:
		GameCxInteraction ();
		virtual ~GameCxInteraction(void){}
		bool AddConnection( qint32 qnSocket);
	signals:
		void signalSendToGServer (int thread_id, String pck);
		void signalSendToThread (int thread_id, String pck);
	private slots:
		void slotSendToGServer (int thread_id, String pck);
		void slotSendToThread (int thread_id, String pck);
		void CloseThread();
};

class GameCxConnectionThread :public QThread
{
	Q_OBJECT
	private:
		int id_;
	public:
		GameCxConnectionThread (qint32 qnSocket, QObject *pParent, const int id_);
		virtual ~GameCxConnectionThread(void);
	protected:
		qint32 m_qnSocket;
		const GameCxInteraction * const m_pInteraction;
		void run(void);
		void removeConnection (GameCxConnection *pConnection);
	signals:
		void signalSendToGServer (int thread_id, String pck);
		void signalSendToThread (String pck);
	private slots:
		void slotSendToGServer (String);
		void slotSendToThread (int, String);
		void connectionError (QAbstractSocket::SocketError socketError);
		void disconnected();
		void destroyed (QObject *pObj = 0);
};

class GameCxConnection : public QTcpSocket
{
	Q_OBJECT
	private:
		int bSize_;
		void write_ (const String & data);
	public:
		GameCxConnection (QObject *pParent = 0);
		virtual ~GameCxConnection(void);
	signals:
		void signalSendToGServer (String pck);
		void readyForUse();
		void newMessage(QString, QString);
	private slots:
		void processReadyRead_ ();
		void slotRecvFromGServer (String pck);
};


#endif
