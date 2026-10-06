#include "../packets.h"

String GamePacket::playerLogin (const String & acc, int uid)
{
	String ret;
	ret += (char)0;
	union b32
	{
		unsigned long word;
		unsigned char byte[4];
	};
	b32 id;
	id.word = uid;
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += id.byte[i];
	}
	for (unsigned int i = 0; i < acc.length (); ++i)
	{
		ret += acc[i];
	}
	while (ret.length () < 56)
	{
		ret += (char)0;
	}
	return ret;
}
