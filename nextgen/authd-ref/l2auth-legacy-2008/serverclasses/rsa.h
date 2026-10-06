#ifndef __RSA_H__
#define __RSA_H__

#include <openssl/rsa.h>
#include "../catx/catxstring.h"

using namespace std;
using namespace CatX;

class MyRSA
{
	private:
		RSA * rsa_;
		String publicKey_;
	public:
		MyRSA ();
		~MyRSA ();
		String publicKey () const;
		String decrypt (const String &);
};

#endif // __RSA_H__
