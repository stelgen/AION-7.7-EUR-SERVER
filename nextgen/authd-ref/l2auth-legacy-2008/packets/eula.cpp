#include "../packets.h"

String Packet::eula ()
{
	String ret;
	// packet id 1
	ret += 0x0b;
	// session 5
	ret += (char) 0;
	ret += 0x0a;
	ret += (char) 0;
	ret += (char) 0;
	// nills 21
	for (unsigned int i = 0; i < 16; ++i)
	{
		ret += (char) 0;
	}
	// some shit 28
	ret += 0xd2;
	ret += 0xcd;
	ret += 0xf5;
	ret += 0x98;
	ret += 0x35;
	ret += 0x14;
	ret += 0x52;
	// cs 32
	ret += checksum_ (ret);
	return ret;
}
