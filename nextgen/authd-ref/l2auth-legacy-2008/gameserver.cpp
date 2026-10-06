#include "gameserver.h"

GameCxServer::GameCxServer (QObject *pParent): QTcpServer( pParent)
{
	if (Data::online ().isOn ())
	{
		pUpdateOnlineTimer_ = new QTimer;
		connect (pUpdateOnlineTimer_, SIGNAL (timeout ()), this, SLOT (slotUpdateOnline_ ()));
		int update = Data::online ().getTimeUpdate ();
		if (update < 60)
		{
			pUpdateOnlineTimer_->start (60000);
		}
		else
		{
			pUpdateOnlineTimer_->start (update * 1000);
		}
	}
	pInteraction_ = new GameCxInteraction();
	connect (pInteraction_, SIGNAL (signalSendToGServer (int, String)), SLOT (slotRecvDataFromThreads (int, String)));
	connect (this, SIGNAL (signalSendToThread (int, String)), pInteraction_, SLOT (slotSendToThread (int, String))); 
}

GameCxServer::~GameCxServer(void)
{
	if (pInteraction_)
	{
		delete pInteraction_;
		pInteraction_ = 0;
	}
	if (pUpdateOnlineTimer_)
	{
		delete pUpdateOnlineTimer_;
		pUpdateOnlineTimer_ = 0;
	}
}

void GameCxServer::slotRecvDataFromServer (int server_id, String pck)
{	
	emit signalSendToThread (idDataGs_.getThreadById (server_id), pck);
}


void GameCxServer::incomingConnection (int nSocket)
{
	pInteraction_->AddConnection (nSocket);
}

void GameCxServer::slotUpdateOnline_ ()
{
	for (vector <Server_s>::const_iterator it = Data::gServerList ().begin (); it != Data::gServerList ().end (); ++it)
	{
		if (it->isOnline ())
		{
			emit signalSendToThread (idDataGs_.getThreadById (it->id ()), GamePacket::ping ());
		}
	}
}

void GameCxServer::slotRecvDataFromThreads (int id, String pck)
{
	int packet_id = pck[0];
	int server_id = idDataGs_.getIdByThread (id);
	String ip = idDataGs_.getIpByThread (id);
	if (static_cast <unsigned char> (packet_id) == GS_CONNECTED)
	{
		ip = pck.substr (1, pck.length ());
		idDataGs_.setIp (id, pck.substr (1, pck.length ()));		
		idDataGs_.setId (id, Data::gServerList ().getServerByIp (ip).id ());
	}
	else if (packet_id == E_GS_PCK_SESSION_KEY2)
	{	
		String ret;
		for (unsigned int i = 1; i < 9; ++i)
		{
			ret += pck[i];
		}
		emit signalSendSessionKey2 (ret);
	}
	else if (packet_id == E_GS_PCK_PLAYER_CONNECTED)
	{
		b32 uid;
		for (unsigned int i = 0; i < 4; ++i)
		{
			uid.byte[i] = pck[i + 1];
		}
		Data::setOnline (uid.word, true, server_id);
		emit signalSqlUserLogin (uid.word, Data::idDataMap ().getIpByUid (uid.word), server_id);
	}
	else if (packet_id == E_GS_PCK_PLAYER_DISCONNECTED)
	{
		String ret;
		b32 uid;
		for (unsigned int i = 0; i < 4; ++i)
		{
			uid.byte[i] = pck[i + 1];
		}
		Data::setOnline (uid.word, false);
		emit signalSqlUserLogout (uid.word);
	}
	else if (packet_id == E_GS_PCK_ONLINE)
	{
		String ret;
		if (pck.length ())
		{
			b32 online;
			for (unsigned int i = 1; i < 3; ++i)
			{
				online.byte[i - 1] = pck[i];
			}
			online.byte[3] = online.byte[2] = 0;
			Data::gServerList ().getServerByIp (ip).setOnline (online.word);			
			emit signalSqlUpdateOnline (ip, online.word);
		}
	}
	else if (packet_id == E_GS_PCK_SERVER_CONNECTED)
	{
		idDataGs_.setId (id, server_id);
		Data::gServerList ().getServerByIp (ip).setIsOnline (true);		
	}
	else if (packet_id == E_GS_PCK_PACKAGE)
	{
		initPackage_ (pck);
		emit signalSendToThread (id, GamePacket::package (pck));
	}
	else if (packet_id == E_GS_PCK_SERVER_DISCONNECTED)
	{
		Data::gServerList ().getServerByIp (ip).setOnline (0);		
		Data::gServerList ().getServerByIp (ip).setIsOnline (false);		
		Data::freeOnline (server_id);
		if (idDataGs_.find (id) != idDataGs_.end ())
		{
			idDataGs_.erase (idDataGs_.find (id));
		}
		emit signalSqlUpdateOnline (ip, 0);
	}
}

void GameCxServer::slotAuthKick (int uid, String account)
{
	Data::setOnline (uid, false);
	for (vector <Server_s>::const_iterator it = Data::gServerList ().begin (); it != Data::gServerList ().end (); ++it)
	{
		if (it->isOnline ())
		{
			emit signalSendToThread (idDataGs_.getThreadById (it->id ()), GamePacket::authKick (uid, account));
		}
	}
}

void GameCxServer::initPackage_ (const String & pck)
{
	unsigned int cnt = Packet::intFromString (pck, 1);
	unsigned int j = 5;
	for (unsigned int i = 0; i < cnt; ++i)
	{
		while (pck[j++] != static_cast <char> (0));
		Data::setOnline (Packet::intFromString (pck, j), true);
		j += 20;
	}
}

