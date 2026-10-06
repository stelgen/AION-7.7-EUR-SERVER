#ifndef BF_H
#define BF_H

#include <stdio.h>
#include <stdlib.h> 
#include "../catx/catxstring.h"
#include <iostream>

class Blowfish
{
	private:
		union b32
		{
  			unsigned long word;
			unsigned char byte[4];
		};		
		unsigned long * P_;
		unsigned long ** S_;		
		void swap_ (b32 *w1, b32 *w2);
		unsigned long F_ (b32 *x) ;		
		void encrypt_ (b32 *w1, b32 *w2);
		void decrypt_ (b32 *w1, b32 *w2);
		void encryptBlock_ (unsigned char *b64);
		void decryptBlock_ (unsigned char *b64);
		int init_;
		CatX::String key_;
	public:
		Blowfish();
		~Blowfish();
		CatX::String checksum (const CatX::String & pck) const;
		int init (const CatX::String & key);
		CatX::String encrypt (const CatX::String & str);
		CatX::String decrypt (const CatX::String & str);
		CatX::String hex2str (const CatX::String & str);
		CatX::String key () const;
};

#endif // BF_H
