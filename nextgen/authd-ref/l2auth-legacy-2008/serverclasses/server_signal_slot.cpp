#include "../server.h"

void CxInteraction::slotSendToThread (int id, String data)
{
	emit signalSendToThread (id, data);
}

void CxInteraction::slotSendToServer (int id, String data)
{
	emit signalSendToServer (id, data);
}


void CxConnectionThread::slotSendToServer (String data)
{
	emit signalSendToServer (id_, data);
}


void CxConnectionThread::slotSendToThread (int id, String data)
{
	if (id == id_)
	{
		emit signalSendToThread (data);
	}
}
