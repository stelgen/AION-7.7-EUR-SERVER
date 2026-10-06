#include "../packets.h"

String GamePacket::package (const String & str)
{
	if (str.length () < 6)
	{
		cout << "Package Error!" << endl;
		exit (1);
	}
	String ret;
	ret += 0x05;
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += str[i + 1];
	}
	for (unsigned int i = 0; i < 11; ++i)
	{
		ret += (char) 0;
	}
	return ret;
}
