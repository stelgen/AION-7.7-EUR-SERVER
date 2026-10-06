#include "../server.h"

CxInteraction::CxInteraction ()
{
	id_ = 0;
	threadCount_ = 0;
}

bool CxInteraction::AddConnection(qint32 qnSocket)
{
	threadMutex_.lock ();
	if (threadCount_ < 200)
	{
		++threadCount_;
		threadMutex_.unlock ();
		return CreateThread (qnSocket) != 0;
	}
	else
	{
		threadMutex_.unlock ();
		return false;
	}
}


CxConnectionThread *CxInteraction::CreateThread( qint32 qnSocket)
{
	CxConnectionThread *pThread = 0;
	try
	{
		pThread = new CxConnectionThread (qnSocket, this, id_++);
		connect (pThread, SIGNAL (finished()), this, SLOT (CloseThread()));
		connect (pThread, SIGNAL (signalSendToServer (int, String)), SLOT (slotSendToServer (int, String)));
		connect (this, SIGNAL (signalSendToThread (int, String)), pThread, SLOT (slotSendToThread (int, String))); 
		pThread->start();
	}
	catch(...)
	{
		if( pThread)
		{
			delete pThread;
			pThread = 0;
		}
	}
	return pThread;
}

void CxInteraction::CloseThread()
{
	if (CxConnectionThread *pThread = qobject_cast<CxConnectionThread *>(sender()))
	{
		threadMutex_.lock ();
		--threadCount_;
		if (!threadCount_)
		{
			Data::idDataMap ().clear ();
		}
		threadMutex_.unlock ();
		pThread->deleteLater();
	}
}

int CxInteraction::threadCount () const
{
	return threadCount_;
}
