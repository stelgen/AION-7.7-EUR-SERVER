#include "../packets.h"

String Packet::authok (const String & a)
{
	String str;
	str += 0x03;
	//test
	str += a;
	for (unsigned int i = 0; i < 8; ++i)
	{
		str += (char) 0;
	}
	str += 0xea;
	str += 0x03;
	for (unsigned int i = 0; i < 8; ++i)
	{
		str += (char) 0;
	}
	str += 0x02;
	for (unsigned int i = 0; i < 27; ++i)
	{
		str += (char) 0;
	}
	str += checksum_ (str);
	for (unsigned int i = 0; i < 4; ++i)
	{
		str += (char) 0;
	}
	//test_end
	return str;
}

