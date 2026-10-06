#include "../gameserver.h"

GameCxInteraction::GameCxInteraction ()
{
	id_ = 0;
}

bool GameCxInteraction::AddConnection(qint32 qnSocket)
{
	return CreateThread (qnSocket) != 0;
}


GameCxConnectionThread *GameCxInteraction::CreateThread( qint32 qnSocket)
{
	GameCxConnectionThread *pThread = 0;
	try
	{
		pThread = new GameCxConnectionThread (qnSocket, this, id_++);
		connect (pThread, SIGNAL (finished()), this, SLOT (CloseThread()));
		connect (pThread, SIGNAL (signalSendToGServer (int, String)), SLOT (slotSendToGServer (int, String)));
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

void GameCxInteraction::CloseThread()
{
	if (GameCxConnectionThread *pThread = qobject_cast<GameCxConnectionThread *>(sender()))
	{
		pThread->deleteLater();
	}
}
