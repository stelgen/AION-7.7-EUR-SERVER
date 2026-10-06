#ifndef _CATX__PARSER_
#define _CATX__PARSER_

#include <map>
#include <vector>
#include <iostream>

#include "catxfunc.h"
#include "catxstring.h"

namespace CatX
{
	class Parser
	{
		private:
			std::map <CatX::String, CatX::String> data_;	
		public:
			Parser () {;}
			Parser (const char * filename);
			CatX::String getString (const CatX::String &);
			int getInt (const CatX::String &);
			double getDouble (const CatX::String &);
			std::vector <CatX::String> getVector (const CatX::String &, const char razdelitel);
	};
}

#endif
