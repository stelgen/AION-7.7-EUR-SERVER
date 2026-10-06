#ifndef __CATX_FUNC_H__
#define __CATX_FUNC_H__

#include "catxstring.h"

namespace CatX
{

	#define ANSI 0
	#define UTF 1

	class Func
	{
		public:
			bool mycmp (const CatX::String &str, const CatX::String &, const unsigned int segm);
			bool mycmp_to_lower (const CatX::String &str, const CatX::String &, const unsigned int segm);
			unsigned int skipComment (const CatX::String & str, const unsigned int i);			
			void cx_write (FILE *f, const char * str, const unsigned int length, int encode);
			void cx_write (FILE *f, const CatX::String & str, int encode);
			CatX::String trim (const CatX::String &);
			CatX::String deleteComment (const CatX::String & str);
	};
}

#endif
