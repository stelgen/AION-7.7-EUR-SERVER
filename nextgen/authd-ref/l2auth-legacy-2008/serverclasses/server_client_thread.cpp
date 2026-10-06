#include "../server.h"

CxConnectionThread::CxConnectionThread (qint32 qnSocket, QObject *pParent, const int id):QThread( pParent), m_pInteraction( (CxInteraction*)pParent)
{
	m_qnSocket = qnSocket;
	id_ = id;
}

CxConnectionThread::~CxConnectionThread(void)
{
}

void CxConnectionThread::disconnected()
{
	if (CxConnection *pConnection = qobject_cast<CxConnection *>(sender()))
	{
	        removeConnection( pConnection);
	}
}

void CxConnectionThread::connectionError( QAbstractSocket::SocketError socketError)
{
	if( CxConnection *pConnection = qobject_cast<CxConnection*>(sender()))
	{
		removeConnection(pConnection);
	}
}

void CxConnectionThread::removeConnection (CxConnection *pConnection)
{
	pConnection->deleteLater();
}

void CxConnectionThread::destroyed( QObject *pObj)
{
	this->exit();
}

void CxConnectionThread::run(void)
{
	try
	{
		CxConnection *pConnection = new CxConnection();
		connect (pConnection, SIGNAL (signalSendToServer (String)), SLOT (slotSendToServer (String)));
		connect (this, SIGNAL (signalSendToThread (String)), pConnection, SLOT (slotRecvFromServer (String))); 
		connect (this, SIGNAL (signalGetIp ()), pConnection, SLOT (slotIp_ ()));
		connect (pConnection, SIGNAL (signalSendIp (QString)), this, SLOT (slotIp_ (QString))); 
		pConnection->setSocketDescriptor (m_qnSocket);
		connect (pConnection, SIGNAL(error(QAbstractSocket::SocketError)), this, SLOT(connectionError(QAbstractSocket::SocketError)));
		connect (pConnection, SIGNAL(disconnected()), this, SLOT(disconnected()));
		connect (pConnection, SIGNAL(destroyed()), this, SLOT(destroyed()));
		emit signalGetIp ();
	}
	catch(...)
	{
		qCritical( "Error in file");
	}
	exec();
}

void CxConnectionThread::slotIp_ (QString ip)
{
	String pck;
	pck += "CONNECTED";
	if (ip.length ())
	{
		pck += ip.toStdString ();
		emit signalSendToServer (id_, pck);
	}
}

CxConnection::CxConnection (QObject *pParent): QTcpSocket( pParent)
{
	QObject::connect(this, SIGNAL (readyRead ()), this, SLOT(processReadyRead_ ()));
	bSize_ = 0;
	pTimer_ = new QTimer;
	connect (pTimer_, SIGNAL (timeout ()), SLOT (slotTimeOut_ ()));
	pTimer_->start (120000);
}

void CxConnection::slotTimeOut_ ()
{
	this->close ();
}

CxConnection ::~CxConnection (void)
{
	delete pTimer_;
}

void CxConnection::processReadyRead_ ()
{
	if (!bSize_ && this->bytesAvailable () >= 2)
	{
		QByteArray ba;
		ba = this->read (2);
		if (Data::server ().getGameGuard () == 0)
		{
			bSize_ = ((unsigned char) ba.at (1)) * 0x100 + ((unsigned char) ba.at (0)) - 2;
		}
		else if (Data::server ().getGameGuard () == 1)
		{
			bSize_ = ((unsigned char) ba.at (0)) * 0x100 + ((unsigned char) ba.at (1)) - 2;
		}
		if (bSize_ <= 0)
		{
			this->close ();
			return;
		}
	}
	if (bSize_ && this->bytesAvailable () >= bSize_)
	{
		QByteArray ba;
		ba = this->read (bSize_);
		String str;
		for (int i = 0; i < ba.size (); ++i)
		{
			str += ba.at (i);
		}
		bSize_ = 0;
		if (str.length ())
		{
			emit signalSendToServer (str);
		}
		processReadyRead_ ();
	}

}

void CxConnection::slotRecvFromServer (String data)
{
	if (data == "CLOSE_CONNECTION")
	{
		this->close ();
	}
	else
	{
		write_ (data);
	}
}

void CxConnection::write_ (const String & data)
{
	String len;
	len += (char) (data.length () + 2) % 0xff;
	len += (char) (data.length () + 2) / 0xff;
	len += data;
	this->write (len.c_str (), len.length ());
}

void CxConnection::slotIp_ ()
{
	emit signalSendIp (this->peerAddress ().toString ());
}




