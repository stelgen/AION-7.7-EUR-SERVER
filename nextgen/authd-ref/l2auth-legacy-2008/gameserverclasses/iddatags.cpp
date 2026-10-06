#include "../gameserver.h"

map <int, pair <int, String> >::const_reverse_iterator IdDataGs::getItById_ (int server_id) const
{
	map <int, pair <int, String> >::const_reverse_iterator it;
	for (it = this->rbegin (); it != this->rend (); ++it)
	{
		if (it->second.first == server_id)
		{
			return it;
		}
	}
	return it;
}

map <int, pair <int, String> >::const_reverse_iterator IdDataGs::getItByIp_ (const String & ip) const
{
	map <int, pair <int, String> >::const_reverse_iterator it;
	for (it = this->rbegin (); it != this->rend (); ++it)
	{
		if (it->second.second == ip)
		{
			return it;
		}
	}
	return it;
}

int IdDataGs::getThreadById (int server_id) const
{
	map <int, pair <int, String> >::const_reverse_iterator it = getItById_ (server_id);
	if (it != this->rend ())
	{
		return it->first;
	}
	else
	{
		return -1;
	}
}

int IdDataGs::getThreadByIp (const String & ip) const
{
	map <int, pair <int, String> >::const_reverse_iterator it = getItByIp_ (ip);
	if (it != this->rend ())
	{
		return it->first;
	}
	else
	{
		return -1;
	}
}

String IdDataGs::getIpById (int server_id) const
{
	map <int, pair <int, String> >::const_reverse_iterator it = getItById_ (server_id);
	if (it != this->rend ())
	{
		return it->second.second;
	}
	else
	{
		return "";
	}	
}

String IdDataGs::getIpByThread (int thread_id) const
{
	map <int, pair <int, String> >::const_reverse_iterator it;
	for (it = this->rbegin (); it != this->rend (); ++it)
	{
		if (it->first == thread_id)
		{
			return it->second.second;
		}
	}
	return "";
}

int IdDataGs::getIdByThread (int thread_id) const
{
	map <int, pair <int, String> >::const_reverse_iterator it;
	for (it = this->rbegin (); it != this->rend (); ++it)
	{
		if (it->first == thread_id)
		{
			return it->second.first;
		}
	}
	return 0;
}

int IdDataGs::getIdByIp (const String & ip) const
{
	map <int, pair <int, String> >::const_reverse_iterator it = getItByIp_ (ip);
	if (it != this->rend ())
	{
		return it->second.first;
	}
	else
	{
		return 0;
	}		
}


