#include "../gameserver.h"

void GameCxInteraction::slotSendToGServer (int id, String data)
{
	emit signalSendToGServer (id, data);
}

void GameCxInteraction::slotSendToThread (int id, String data)
{
	emit signalSendToThread (id, data);
}


void GameCxConnectionThread::slotSendToGServer (String data)
{
	emit signalSendToGServer (id_, data);
}


void GameCxConnectionThread::slotSendToThread (int id, String data)
{
	if (id == id_)
	{
		emit signalSendToThread (data);
	}
}
