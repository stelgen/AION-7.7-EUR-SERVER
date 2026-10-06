#include "packets.h"

String Packet::checksum_ (const String & raw)
{
	b32 chksum;
	chksum.word = 0;
	unsigned char * a = new unsigned char [raw.length ()];
	for (unsigned int i = 0; i < raw.length (); ++i)
	{
		a[i] = raw[i];
	}
	for (unsigned int i = 0; i < raw.length (); i += 4 )
	{
		chksum.word ^= *((unsigned long *)&a[i]);
	}
	delete [] a;
	String ret;
	int j = 4;
	while (1)
	{
		if (j == 0)
		{
			break;
		}
		else
		{
			if ((char)chksum.byte [j - 1] == -1)
			{
				ret += (char) 0;
				--j;
			}
			else
			{
				break;
			}
		}
	}
	for (int i = 0; i < j; ++i)
	{
		ret += chksum.byte [i];
	}
	return ret;
}

String Packet::dword_ (const int a)
{
	String ret;
	ret += a % 0x100;
	ret += (a / 0x100) % 0x100;
	return ret;
}

String Packet::ip_ (const String & ip)
{
	String digits[4];
	int j = 0;
	String dig;
	for (unsigned int i = 0; i < ip.length (); ++i)
	{
		if (ip[i] == '.')
		{
			if (j > 2)
			{
				break;
			}
			else
			{
				digits[j++] = dig;
				dig.clear ();
			}
		}
		else
		{
			dig += ip[i];
		}
	}
	digits[j] = dig;
	int conv[4];
	String ret;
	for (unsigned int i = 0; i < 4; ++i)
	{
		sscanf (digits[i].c_str (), "%d", &conv[i]);
		ret += static_cast <char> (conv[i]);
	}
	return ret;
}

String Packet::scrambleRSA_ (const String & str)
{
	String ret = str;
	if (str.length () == 128)
	{
		for (unsigned int i = 0; i < 4; ++i)
		{
			unsigned char temp = ret[0x00 + i];
			ret[0x00 + i] = ret[0x4d + i];
			ret[0x4d + i] = temp;
		}
		for (unsigned int i = 0; i < 64; ++i)
		{
			ret[i] = (unsigned char)(ret[i] ^ ret[0x40 + i]);
		}
		for (unsigned int i = 0; i < 4; ++i)
		{
			ret[0x0d + i] = (unsigned char)(ret[0x0d + i] ^ ret[0x34 + i]);
		}
		for (unsigned int i = 0; i < 64; ++i) 
		{
			ret[0x40 + i] = (unsigned char)(ret[0x40 + i] ^ ret[i]);
		}
		return ret;
	}
	else
	{
		return "";
	}
}

bool Packet::checkPckSum (const Blowfish & bfish, const String & pck)
{
	unsigned int segment = 0;
	switch (pck[0])
	{
		case E_PCK_REQUEST_AUTH_LOGIN:
			if (Data::server ().getRevision () == REVISION_c621 && pck.length () > 160)
			{
				segment = E_CS_REQUEST_AUTH_LOGIN_CT1;
			}
			else 
			{
				segment = E_CS_REQUEST_AUTH_LOGIN_C4;
			}
			break;
		case E_PCK_REQUEST_SERVER_LOGIN:
			segment = E_CS_REQUEST_SERVER_LOGIN_CT1;
			break;
		case E_PCK_REQUEST_SERVER_LIST:
			segment = E_CS_REQUEST_SERVER_LIST_CT1;
			break;
	}
	if (pck.length () > segment)
	{
		if (bfish.checksum (pck.substr (0, segment)) == pck.substr (segment, 4))
		{
			return true;
		}
		else
		{			
			return false;
		}
	}
	else
	{
		return false;
	}
}

int Packet::intFromString (const String & src, int pos)
{
	b32 a;
	for (unsigned int i = 0; i < 4; ++i)
	{
		a.byte[i] = src.at (pos + i);
	}
	return a.word;
}
