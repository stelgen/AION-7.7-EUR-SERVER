#include "authmainwindow.h"

AuthMainWindow::AuthMainWindow (const QString & v, const QString & b)
{
	// update Timer
	timer_ = time (0);
	timeTimer_ = new QTimer;
	connect (timeTimer_, SIGNAL (timeout ()), SLOT (slotTimeOut_ ()));
	timeTimer_->start (1000);

	// widget
	timeLabel_ = new QLabel ("Uptime: 0s");
	this->setCentralWidget (timeLabel_);
	this->resize (200, 100);
	this->setWindowTitle ("CatX Auth v. " + v + " Build " + b);

	// metatypes
	qRegisterMetaType <String> ("String");
	qRegisterMetaType <QAbstractSocket::SocketError> ("QAbstractSocket::SocketError");

	// parser
	Parser * myP = new Parser ("auth.ini");
	Data::initData (myP);
	delete myP;

	// sql
	pSql_ = new MySQL ();
	connect (this, SIGNAL (signalInitServerList ()), pSql_, SLOT (slotInitServerList ()));
	connect (pSql_, SIGNAL (signalInitServerListComplete ()), this, SLOT (slotInitServers ()));
	pSql_->start ();
	emit signalInitServerList ();
}

void AuthMainWindow::slotInitServers ()
{
	// gameServer
	int port = Data::server ().getIntPort ();
	GameCxServer * pGameServer = new GameCxServer ();
	connect (pGameServer, SIGNAL (signalSqlUpdateOnline (const String &, const int)), pSql_, SLOT (slotUpdateOnline (const String &, const int)));
	connect (pGameServer, SIGNAL (signalSqlUserLogin (const int, const String &, int)), pSql_, SLOT (slotUserLogin (const int, const String &, int)));
	connect (pGameServer, SIGNAL (signalSqlUserLogout (const int)), pSql_, SLOT (slotUserLogout (const int)));

	if (!pGameServer->listen (QHostAddress::Any, port))
	{                          
		String err = "IntPort in use: ";
		err.append (port);
		Error::writeError (err);
		exit (1);
	}

	// server
	CxServer * server = new CxServer ();
	connect (pGameServer, SIGNAL (signalSendSessionKey2 (String)), server, SLOT (slotRecvSessionKey2FromGS (String)));
	connect (server, SIGNAL (sendDataToGs (int, String)), pGameServer, SLOT (slotRecvDataFromServer (int, String)));

	connect (server, SIGNAL (signalAuthKick (int, String)), pGameServer, SLOT (slotAuthKick (int, String)));

	connect (server, SIGNAL (signalSqlCheckLogin (const String &, int, const String &, const String &)), pSql_, SLOT (slotCheckLogin (const String &, int, const String &, const String &)));
	connect (server, SIGNAL (signalSqlLogLogin (const String &, const String &, const String &)), pSql_, SLOT (slotLogLogin (const String &, const String &, const String &)));
	connect (pSql_, SIGNAL (signalCheckLogin (const String &, int, int, int, int)), server, SLOT (slotCheckLogin (const String &, int, int, int, int)));

	port = Data::server ().getExPort ();

	if (!server->listen (QHostAddress::Any, port))
	{
		String err = "ExPort in use: ";
		err.append (port);
		Error::writeError (err);
		exit (1);
	}
}

AuthMainWindow::~AuthMainWindow ()
{
	delete timeTimer_;
	delete timeLabel_;
}

void AuthMainWindow::slotTimeOut_ ()
{
	QString str;
	str += "Uptime: ";
	str += String (static_cast <int> (time (0) - timer_)).c_str ();
	str += "s";
	timeLabel_->setText (str);
}

