#include "../packets.h"

String GamePacket::authKick (int uid, const String & acc)
{
	String ret;
	ret += 0x01;
	b32 buid;
	buid.word = uid;
	for (unsigned int i = 0; i < 4; ++i)	
	{
		ret += buid.byte[i];
	}
	ret += acc;
	ret += static_cast <char> (0);
	return ret;
}
