#include "../packets.h"

String Packet::initC4 ()
{
	String ret;
	ret += (char) 0;
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	ret += 0x5a;
	ret += 0x78;
	ret += (char) 0;
	ret += (char) 0;
	return ret;
}

String Packet::init (const String & key, const String & bf)
{
	String ret;
	// Packet ID 1
	ret += (char) 0;
	// SESSION 5
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	ret += static_cast <char> (rand () % 0x100);
	// REVISION 9
	ret += 0x21;
	ret += 0xc6;
	ret += (char) 0;
	ret += (char) 0;
	// RSA 137
	ret += scrambleRSA_ (key);
	// NULLS 153
	for (unsigned int i = 0; i < 16; ++i)
	{
		ret += (char) 0;
	}
	// BLOWFISH 169
	ret += bf;
	// +7 digits 176
	for (unsigned int i = 0; i < 7; ++i)
	{
		ret += (char) 0;
	}
	String xr;
	xr = "asdf";
	union b32
	{
		unsigned long word;
		unsigned char byte[4];
	};
	b32 s, k;
	for (unsigned int i = 0; i < 4; ++i)
	{
		k.byte[i] = xr[i];
	}
	s.word = 0;
	for (unsigned int j = 4; j < ret.length () - 3; j += 4)
	{
		for (unsigned int i = j; i < j + 4; ++i)
		{
			s.byte[i-j] = ret[i];
		}
		k.word += s.word;
		s.word ^= k.word;
		for (unsigned int i = j; i < j + 4; ++i)
		{
			ret[i] = s.byte[i-j];
		}
	}
	// XOR 180
	for (unsigned int i = 0; i < 4; ++i)
	{
		ret += k.byte[i];
	}
	// CHECHSUM 184
	ret += checksum_ (ret);
	return ret;
}
