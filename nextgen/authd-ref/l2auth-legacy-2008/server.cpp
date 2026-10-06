#include "server.h"

CxServer::CxServer (QObject *pParent): QTcpServer( pParent)
{
	srand (time (0));
	for (unsigned int i = 0; i < RSA_COUNT; ++i)
	{
		pRsa_[i] = new MyRSA;
	}
	pSecurity_ = new Security (Data::server ().getMaxThreads (), "black.txt");
	pSecurity_->loadBlackList ();
	pInteraction_ = new CxInteraction();
	connect (pInteraction_, SIGNAL (signalSendToServer (int, String)), SLOT (slotRecvFromThread (int, String)));
	connect (this, SIGNAL (signalSendToThread (int, String)), pInteraction_, SLOT (slotSendToThread (int, String))); 

	bfish_.init (bfish_.hex2str (Data::server ().getBlowfish ()));
}

CxServer::~CxServer(void)
{
	if (pInteraction_)
	{
		delete pInteraction_;
		pInteraction_ = 0;
	}
	if (pSecurity_)
	{
		delete pSecurity_;
		pSecurity_ = 0;
	}
	if (*pRsa_)
	{
		for (unsigned int i = 0; i < RSA_COUNT; ++i)
		{
			delete pRsa_[i];
		}
		*pRsa_ = 0;
	}
}

void CxServer::slotRecvFromThread (int id, String packet)
{	
	if (!packet.length ())
	{
		Error::writeError ("CxServer::slotRecvFromThread zero length. This saved server");
		return;
	}
	if (packet.substr (0, 9) != "CONNECTED")
	{
		String pck = bfish_.decrypt (packet);
		int packet_id = pck[0];
		if (packet_id == E_PCK_REQUEST_AUTH_LOGIN)
		{
			if (Packet::checkPckSum (bfish_, pck))
			{
				requestAuthLogin_ (id, pck, Data::idData (id).getIp ());
			}
			else
			{
				emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));	
			}			
		}
		else if (packet_id == E_PCK_REQUEST_SERVER_LOGIN)
		{
			if (Packet::checkPckSum (bfish_, pck))
			{
				if (Data::gServerList ().isOnline (pck[SERVER_SELECTION_SEGM]))
				{
					if (Data::idData (id).getAccountName ().length ())
					{		
						emit signalSqlLogLogin (Data::idData (id).getAccountName (), Data::idData (id).getIp (), Data::gServerList ().innerIp (pck[SERVER_SELECTION_SEGM]));
						emit sendDataToGs (pck[SERVER_SELECTION_SEGM], GamePacket::playerLogin (Data::idData (id).getAccountName (), Data::idData (id).getUid ()));
					}
					else
					{
						emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));
					}                         
				}
				else
				{
					emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));
				}
			}
			else
			{
				emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));	
			}			
		}
		else if (packet_id == E_PCK_REQUEST_SERVER_LIST)
		{
			if (Packet::checkPckSum (bfish_, pck))
			{
				emit signalSendToThread (id, bfish_.encrypt (Packet::serverList (Data::idData (id).getLastWorld ())));	
			}
			else
			{
				emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));
			}
		}
		else if (packet_id == E_PCK_EULA)
		{
			emit signalSendToThread (id, bfish_.encrypt (Packet::eula ()));
		}
		else
		{
			emit signalSendToThread (id, bfish_.encrypt (Packet::playFail ()));
			pSecurity_->ban (Data::idData (id).getIp (), E_SECURITY_INVALID_PACKET);
		}
	}
	else
	{
		String ip = packet.substr (9, packet.length ());
		if (
			(!pSecurity_->maxThreads () && pInteraction_->threadCount () >= pSecurity_->maxThreads ()) ||
			pSecurity_->isBlackIp (ip) ||
			!pSecurity_->checkTime (ip)
		)
		{
			pSecurity_->ban (ip, E_SECURITY_BRUT);
			emit signalSendToThread (id, "CLOSE_CONNECTION");
		}
		else
		{
			pSecurity_->initTime (ip);
			Data::idData (id).setIp (ip);
			int revision = Data::server ().getRevision ();
			if (revision == REVISION_c621)
			{
				Data::idData (id).setRsa (rand () % RSA_COUNT);
				emit signalSendToThread (id, bfish_.encrypt (Packet::init (pRsa_[Data::idData (id).getRsa ()]->publicKey (), bfish_.key ())));
			}
			else if (revision == REVISION_785a)
			{	
				emit signalSendToThread (id, Packet::initC4 ());
			}
		}
	}
}


void CxServer::incomingConnection (int nSocket)
{
	pInteraction_->AddConnection (nSocket);
}

void CxServer::requestAuthLogin_ (const int id, const String & packet, const String & ip)
{
	if (packet.length () > 30)
	{
		String pck = packet.substr (1, packet.length ());
		if (packet.length () > 128)
		{
		// C6+
			pck = pRsa_[Data::idData (id).getRsa ()]->decrypt (packet.substr (1, 128));
			unsigned int i = 0;
			String brk;
			brk += 0x24;
			brk += (char) 0;
			brk += (char) 0;
			for (i = 0; i < pck.length () - 4; ++i)
			{
				if (pck.substr (i, 3) == brk)
				{
					break;
				}
			}
			pck = pck.substr (i + 3, 30);
			if (pck.length () < 30)
			{
				emit signalSendToThread (id, bfish_.encrypt (Packet::authfail (SYSTEM_ERROR)));
				return;
			}
		}
		String login = pck.substr (0, 14);
		String pass = pck.substr (14, 16);
		emit signalSqlCheckLogin (login, id, pass, ip);
	}
	else
	{
		emit signalSendToThread (id, bfish_.encrypt (Packet::authfail (SYSTEM_ERROR)));
		return;
	}
}

void CxServer::slotRecvSessionKey2FromGS (String pck)
{	
	b32 session2;
	String serverIp;
	for (unsigned int i = 0; i < 4; ++i)
	{
		session2.byte[i] = pck[i];
	}
	emit signalSendToThread (Data::idDataMap ().getIdByUid (session2.word), bfish_.encrypt (Packet::playOk (pck.substr (0, 8))));
}

void CxServer::slotCheckLogin (String login, int id, int uid, int last_world, int request)
{
	String sessionKey1;
	String newLogin;
	for (unsigned int i = 0; i < 14; ++i)
	{
		if (login[i] == (char) 0)
		{
			break;
		}
		newLogin += login [i];
	}
	// 0 = ok, 1-7 errors
	if (!request)
	{      	
		pSecurity_->clearIllegal (Data::idData (id).getIp ());
		Data::idData (id).setUid (uid);
		Data::idData (id).setLastWorld (last_world);
		for (unsigned int i = 0; i < 8; ++i)
		{
			sessionKey1 += rand () % 256;
		}
		Data::idData (id).setAccountName (newLogin);
		emit signalSendToThread (id, bfish_.encrypt (Packet::authok (sessionKey1)));
	}
	else
	{
		switch (request)
		{
			case WRONG_IP:
				request = ACCESS_FORBIDDEN;
				break;
			case INVALID_PASSWORD:
				pSecurity_->incIllegal (Data::idData (id).getIp ());
				pSecurity_->ban (Data::idData (id).getIp (), E_SECURITY_INVALID_PASSWORD);
				break;
			case ACCOUNT_IN_USE:
				emit signalAuthKick (uid, login);
				break;		
		}
		emit signalSendToThread (id, bfish_.encrypt (Packet::authfail (request)));
	}
}

