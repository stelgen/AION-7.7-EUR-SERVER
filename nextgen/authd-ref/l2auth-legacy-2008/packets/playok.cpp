#include "packets.h"

String Packet::playOk (const String & str)
{
	String ret;
	ret += 0x07;
	if (str.length () > 4)
	{
		ret += str.substr (4, 4);
		ret += str.substr (0, 4);
	}
	ret += checksum_ (ret);
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
