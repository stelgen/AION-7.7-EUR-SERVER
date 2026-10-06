#include "rsa.h"

MyRSA::MyRSA ()
{
	rsa_ = RSA_new ();
	rsa_ = RSA_generate_key (1024, 65537, NULL, NULL);
	unsigned char * key = new unsigned char [129];
	BN_bn2bin (rsa_->n, key);
	publicKey_.clear ();
	for (unsigned int i = 0; i < 128; ++i)
	{
		publicKey_ += key[i];
	}
	delete [] key;
}

MyRSA::~MyRSA ()
{
	RSA_free (rsa_);
}

String MyRSA::publicKey () const
{
	return publicKey_;
}

String MyRSA::decrypt (const String & str)
{
	unsigned char * from = new unsigned char [str.length () + 1];
	for (unsigned int i = 0; i < str.length (); ++i)
	{
		from[i] = (unsigned char) str[i];
	}
	unsigned char * to = new unsigned char [128 * (1 + (int)((double)str.length () / 128))];
	if (RSA_private_decrypt (str.length (), from, to, rsa_, RSA_NO_PADDING) != 128)
	{
		return "";
	}
	String ret;
	for (unsigned int i = 0; i < str.length (); ++i)
	{
		ret += (char) to[i];
	}
	delete [] from;
	delete [] to;
	return ret;
}


