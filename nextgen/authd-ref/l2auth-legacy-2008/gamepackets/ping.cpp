#include "../packets.h"

String GamePacket::ping ()
{
	String ret;
	ret += 0x02;
	for (unsigned int i = 0; i < 7; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
