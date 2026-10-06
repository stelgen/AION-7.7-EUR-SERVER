#include "../packets.h"

String Packet::playFail ()
{
	String ret;
	ret += 0x06;
	for (unsigned int i = 0; i < 3; ++i)
	{
		ret += (char) 0;
	}
	ret += 0x04;
	ret += checksum_ (ret);
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
