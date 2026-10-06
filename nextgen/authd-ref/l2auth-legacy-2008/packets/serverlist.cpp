#include "../packets.h"

String Packet::serverList (int lastworld)
{
	String ret;
	ret += 0x04;
	int serverAmount = Data::gServerList ().size ();
	ret += static_cast <char> (serverAmount);
	ret += static_cast <char> (lastworld);
	for (vector <Server_s>::const_iterator i = Data::gServerList ().begin (); i != Data::gServerList ().end (); ++i)
	{
		ret += i->id ();
		ret += ip_ (i->ip ());
		b32 uid;
		uid.word = i->port ();
		for (unsigned int j = 0; j < 4; ++j)
		{
			ret += uid.byte[j];
		}
		ret += i->ageLimit ();//slist[i].ageLimit;
		ret += 0x01;
		ret += dword_ (0);
		ret += dword_ (5000);
		ret += dword_ (i->isOnline ());
		for (unsigned int j = 0; j < 4; ++j)
		{
			ret += static_cast <char> (0);
		}
	}
	ret += checksum_ (ret);
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
