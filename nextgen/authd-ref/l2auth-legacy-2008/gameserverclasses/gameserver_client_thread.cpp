#include "../gameserver.h"

GameCxConnectionThread::GameCxConnectionThread (qint32 qnSocket, QObject *pParent, const int id):QThread( pParent), m_pInteraction( (GameCxInteraction*)pParent)
{
	m_qnSocket = qnSocket;
	id_ = id;
}

GameCxConnectionThread::~GameCxConnectionThread(void)
{
}

void GameCxConnectionThread::disconnected()
{
	String str;
	str += E_GS_PCK_SERVER_DISCONNECTED;
	emit signalSendToGServer (id_, str);
	if (GameCxConnection *pConnection = qobject_cast<GameCxConnection *>(sender()))
        removeConnection( pConnection);
}

void GameCxConnectionThread::connectionError( QAbstractSocket::SocketError socketError)
{
	if( GameCxConnection *pConnection = qobject_cast<GameCxConnection*>(sender()))
	{
		removeConnection(pConnection);
	}
}

void GameCxConnectionThread::removeConnection (GameCxConnection *pConnection)
{
	pConnection->deleteLater();
}

void GameCxConnectionThread::destroyed( QObject *pObj)
{
	this->exit();
}

void GameCxConnectionThread::run(void)
{
	try
	{
		GameCxConnection *pConnection = new GameCxConnection;
		connect (pConnection, SIGNAL (signalSendToGServer (String)), SLOT (slotSendToGServer (String)));
		connect (this, SIGNAL (signalSendToThread (String)), pConnection, SLOT (slotRecvFromGServer (String))); 
		pConnection->setSocketDescriptor (m_qnSocket);
		connect (pConnection, SIGNAL(error(QAbstractSocket::SocketError)), this, SLOT(connectionError(QAbstractSocket::SocketError)));
		connect (pConnection, SIGNAL(disconnected()), this, SLOT(disconnected()));
		connect (pConnection, SIGNAL(destroyed()), this, SLOT(destroyed()));
		String pck;
		pck += GS_CONNECTED;
		pck += pConnection->peerAddress ().toString ().toStdString ();
		emit signalSendToGServer (id_, pck);
	}
	catch(...)
	{
		qCritical( "Error in file");
	}
	exec();
}

GameCxConnection::GameCxConnection (QObject *pParent): QTcpSocket( pParent)
{
	QObject::connect(this, SIGNAL (readyRead ()), this, SLOT(processReadyRead_ ()));
	bSize_ = 0;
}

GameCxConnection ::~GameCxConnection (void)
{
}

void GameCxConnection::processReadyRead_ ()
{
	if (!bSize_ && this->bytesAvailable () >= 2)
	{
		QByteArray ba;
		ba = this->read (2);
		bSize_ = ((unsigned char) ba.at (1)) * 0x100 + ((unsigned char) ba.at (0)) - 2;
	}
	if (bSize_ && this->bytesAvailable () >= bSize_)
	{
		QByteArray ba;
		ba = this->read (bSize_);
		String str;
		for (unsigned int i = 0; i < static_cast <unsigned int> (ba.size ()); ++i)
		{
			str += ba.at (i);
		}
		bSize_ = 0;
		if (ba.size ())
		{
			emit signalSendToGServer (str);
		}
		processReadyRead_ ();
	}

}

void GameCxConnection::slotRecvFromGServer (String data)
{
	write_ (data);
}

void GameCxConnection::write_ (const String & data)
{
	String len;
	len += (char) (data.length () + 2) % 0xff;
	len += (char) (data.length () + 2) / 0xff;
	len += data;
	this->write (len.c_str (), len.length ());
}


