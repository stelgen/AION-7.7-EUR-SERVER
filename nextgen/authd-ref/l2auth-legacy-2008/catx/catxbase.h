#ifndef __CATXBASE__
#define __CATXBASE__

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

namespace CatX
{
	class CatXBase
	{
		private:
			static bool debugMode_;
			static bool errorMode_;
		public:
			CatXBase ();
			static void setDebugMode (bool mode);
			static void setErrorMode (bool mode);
			void catxError (const char * msg) const;
			void catxError (const char * msg, const char * err) const;
			void catxError (const char * msg, const char * err, const int len) const;
			void catxError (const char * msg, const int val) const;
			void catxDebug (const char * msg) const;
			void catxDebug (const char * msg, const char * err) const;
			void catxDebug (const char * msg, const char * err, const int len) const;
			void catxDebug (const char * msg, const int val) const;
	};
};

#endif
