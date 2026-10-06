#ifndef __CATXSTRING__
#define __CATXSTRING__

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <math.h>
#include <limits.h>
#include <string>

#include "catxbase.h"

namespace CatX
{
	enum
	{
		E_OK = 0,
		E_INVALID_NUMBER = 1,
		E_OVERFLOW = 2
	};

	class String : public CatX::CatXBase
	{
		private:
			char * data_;
			unsigned int len_;
			unsigned int alloc_;
			void allocate_ (unsigned int len);
		public:
			String ();
			String (char);
			String (const char * str);
			String (const int val);
			String (const double val);
			String (const char * str, unsigned int len);
			String (const std::string & el);
			String (const String & el);
			~String ();
			unsigned int length () const;
			char * c_str () const;
			char * c_str (unsigned int pos) const;
			void clear ();
			String operator + (const char c) const;
			String operator + (const char * str) const;
			String operator + (const String & el) const;
			String & operator += (const char c);
			String & operator += (const char * str);
			String & operator += (const String & el);
			String & operator += (const std::string & el);
			bool operator == (const String & el) const;
			bool operator == (const std::string & el) const;
			bool operator == (const char * str) const;
			bool operator != (const String & el) const;
			bool operator != (const std::string & el) const;
			bool operator != (const char * str) const;
			bool operator < (const String & el) const;
			bool operator < (const char * str) const;
			String substr (const unsigned int pos, const unsigned int len) const;
			void append (const char * str, const unsigned int len);
			void append (const char * str);
			void append (const char c);
			void append (const String & el);
			void append (const int el);
			void append (const double el);
			void operator = (const String & el);
			void operator = (const std::string & el);
			void operator = (const char * el);
			char & operator [] (const unsigned int);
			char operator [] (const unsigned int) const;
			char at (const unsigned int) const;
			void freeEx ();
			int toInt () const;
			double toDouble () const;
			String windows_str () const;
			String toUtf () const;
	};
};

#endif
