#include "../server.h"

map <int, IdData>::const_reverse_iterator IdDataMap::getIteratorByUid_ (int uid) const
{
	map <int, IdData>::const_reverse_iterator it;
	for (it = this->rbegin (); it != this->rend (); ++it)
	{
		if (it->second.getUid () == uid)
		{
			break;
		}
	}
	return it;
}


int IdDataMap::getIdByUid (int uid) const
{
	map <int, IdData>::const_reverse_iterator it = getIteratorByUid_ (uid);
	if (it != this->rend ())
	{
		return it->first;
	}
	else 
	{
		return 0;
	}
}

String IdDataMap::getIpByUid (int uid) const
{
	map <int, IdData>::const_reverse_iterator it = getIteratorByUid_ (uid);
	if (it != this->rend ())
	{
		return it->second.getIp ();
	}
	else 
	{
		return "";	
	}
}

String IdDataMap::getAccountByUid (int uid) const
{
	map <int, IdData>::const_reverse_iterator it = getIteratorByUid_ (uid);
	if (it != this->rend ())
	{
		return it->second.getAccountName ();
	}
	else 
	{
		return "";	
	}
}

