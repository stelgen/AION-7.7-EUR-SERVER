#ifndef __SERVER__
#define __SERVER__

#define RSA_COUNT 10
#define SERVER_SELECTION_SEGM 9

#include <QtNetwork>
#include <QMutex>

#include "packets.h"
#include "gameserver.h"
#include "serverclasses/mysql.h"
#include "serverclasses/bfish.h"
#include "serverclasses/rsa.h"
#include "serverclasses/security.h"
#include "serverclasses/iddatamap.h"	

#include "catx/parser.h"
#include "catx/catxstring.h"

using namespace CatX;
using namespace std;

class CsServer;
class CxInteraction;
class CxConnectionThread;
class CxConnection;

class CxServer :public QTcpServer
{
	Q_OBJECT
	private:
		CxInteraction * pInteraction_;
		Security * pSecurity_;
		MyRSA * pRsa_ [RSA_COUNT];
		Blowfish bfish_;
		void requestAuthLogin_ (const int id, const String & ip, const String& pck);				
	public:
		CxServer (QObject *pParent = 0);
		virtual ~CxServer(void);
	signals:
		void signalSendToServer (int thread_id, String pck);
		void signalSendToThread (int thread_id, String pck);
		void sendDataToGs (int server_id, String pck);
		void signalSqlCheckLogin (String, int, String, String);
		void signalSqlLogLogin (String, String, String);
		void signalAuthKick (int, String);
	private slots:
		void slotRecvFromThread (int thread_id, String pck);
		void slotRecvSessionKey2FromGS (String);
		void slotCheckLogin (String, int, int, int, int);
	protected:
		void incomingConnection (int nSocket);
};

class CxInteraction: public QObject
{
	Q_OBJECT
	private:
		int id_;
		int threadCount_;
		QMutex threadMutex_;
		CxConnectionThread *CreateThread( qint32 qnSocket);
	public:
		CxInteraction ();
		virtual ~CxInteraction(void){}
		bool AddConnection( qint32 qnSocket);
		int threadCount () const;
	signals:
		void signalSendToServer (int thread_id, String pck);
		void signalSendToThread (int thread_id, String pck);
	private slots:
		void slotSendToServer (int thread_id, String pck);
		void slotSendToThread (int thread_id, String pck);
		void CloseThread();
};

class CxConnectionThread :public QThread
{
	Q_OBJECT
	private:
		int id_;
	public:
		CxConnectionThread (qint32 qnSocket, QObject *pParent, const int id_);
		virtual ~CxConnectionThread(void);
	protected:
		qint32 m_qnSocket;
		const CxInteraction * const m_pInteraction;
		void run(void);
		void removeConnection (CxConnection *pConnection);
	signals:
		void signalSendToServer (int thread_id, String pck);
		void signalSendToThread (String pck);
		void signalGetIp ();
	private slots:
		void slotSendToThread (int thread_id, String pck);
		void slotSendToServer (String pck);
		void connectionError (QAbstractSocket::SocketError socketError);
		void disconnected();
		void destroyed (QObject *pObj = 0);
		void slotIp_ (QString);
};

class CxConnection : public QTcpSocket
{
	Q_OBJECT
	private:
		QTimer * pTimer_;
		int bSize_;
		QByteArray ba_;
		String login_;
		void write_ (const String & data);
	public:
		CxConnection (QObject *pParent = 0);
		virtual ~CxConnection(void);
	signals:
		void signalSendToServer (String pck);
		void readyForUse();
		void newMessage(QString, QString);
		void signalSendIp (QString);
		void signalAuthKick (int uid, String account);
	private slots:
		void processReadyRead_ ();
		void slotRecvFromServer (String pck);
		void slotTimeOut_ ();
		void slotIp_ ();
};


#endif
