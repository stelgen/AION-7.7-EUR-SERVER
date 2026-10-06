#include "../packets.h"

String Packet::authfail (const int reason)
{
	String ret;
	ret += 0x01;
	ret += (char) reason % 256;
	ret += (char) 0;
	ret += (char) 0;
	ret += (char) 0;
	ret += checksum_ (ret);
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
