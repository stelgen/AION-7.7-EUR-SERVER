#include "serverlist.h"

bool ServerVector::isOnline (int id) const
{
	for (vector <Server_s>::const_iterator p = ServerVector::begin (); p != ServerVector::end (); ++p)
	{
		if (p->id () == id)
		{
			return p->isOnline ();
		}
	}
	return false;
}

void ServerVector::init (const vector <Server_s> & data)
{
	for (vector <Server_s>::const_iterator it = data.begin (); it != data.end (); ++it)
	{
		ServerVector::push_back (*it);
	}
}

String ServerVector::innerIp (int id) const
{
	for (vector <Server_s>::const_iterator it = ServerVector::begin (); it != ServerVector::end (); ++it)
	{
		if (it->id () == id)
		{
			return it->innerIp ();
		}
	}
	return "0.0.0.0";
}

Server_s & ServerVector::getServerByIp (const String & ip)
{
	vector <Server_s>::iterator it;
	for (it = ServerVector::begin (); it != ServerVector::end (); ++it)
	{
		if (it->innerIp () == ip)
		{
			return *it;
		}
	}
	return *it;
}
