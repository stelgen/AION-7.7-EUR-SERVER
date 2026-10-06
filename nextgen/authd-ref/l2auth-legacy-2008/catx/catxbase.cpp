#include "catxbase.h"

#define ERROR_FILE_NAME "catxlib_error.log"
#define DEBUG_FILE_NAME "catxlib_warning.log"

bool CatX::CatXBase::debugMode_ = false;
bool CatX::CatXBase::errorMode_ = false;

CatX::CatXBase::CatXBase ()
{
}

void CatX::CatXBase::catxError (const char * msg) const
{
	if (strlen (ERROR_FILE_NAME))
	{
		if (errorMode_)
		{
			FILE *f = fopen (ERROR_FILE_NAME, "ab");
			fprintf (f, "%s\n", msg);
			fclose (f);
		}
	}
}

void CatX::CatXBase::catxError (const char * msg, const char * str) const
{
	if (strlen (ERROR_FILE_NAME))
	{
		if (errorMode_)
		{
			FILE *f = fopen (ERROR_FILE_NAME, "ab");
			fprintf (f, "%s %s\n", msg, str);
			fclose (f);
		}
	}
}


void CatX::CatXBase::catxError (const char * msg, const char * str, const int len) const
{
	if (strlen (ERROR_FILE_NAME))
	{
		if (errorMode_)
		{
			FILE *f = fopen (ERROR_FILE_NAME, "ab");
			fprintf (f, "%s ", msg);
			for (int i = 0; i < len; ++i)
			{
				fprintf (f, "%c", str[i]);
			}
			fprintf (f, "\n");
			fclose (f);
		}
	}
}

void CatX::CatXBase::catxError (const char * msg, const int val) const
{
	if (strlen (ERROR_FILE_NAME))
	{
		if (errorMode_)
		{
			FILE *f = fopen (ERROR_FILE_NAME, "ab");
			fprintf (f, "%s %d\n", msg, val);
			fclose (f);
		}
	}
}

void CatX::CatXBase::setDebugMode (bool mode)
{
	debugMode_ = mode;
}

void CatX::CatXBase::setErrorMode (bool mode)
{
	errorMode_ = mode;
}


void CatX::CatXBase::catxDebug (const char * msg) const
{
	if (strlen (DEBUG_FILE_NAME))
	{
		if (debugMode_)
		{
			FILE *f = fopen (DEBUG_FILE_NAME, "ab");
			fprintf (f, "%s\n", msg);
			fclose (f);
		}
		else
		{
			catxError (msg);
			exit (1);
		}
	}
}

void CatX::CatXBase::catxDebug (const char * msg, const char * str) const
{
	if (strlen (DEBUG_FILE_NAME))
	{
		if (debugMode_)
		{
			FILE *f = fopen (DEBUG_FILE_NAME, "ab");
			fprintf (f, "%s %s\n", msg, str);
			fclose (f);
		}
		else
		{
			catxError (msg, str);
			exit (1);
		}
	}
}


void CatX::CatXBase::catxDebug (const char * msg, const char * str, const int len) const
{
	if (strlen (DEBUG_FILE_NAME))
	{
		if (debugMode_)
		{
			FILE *f = fopen (DEBUG_FILE_NAME, "ab");
			fprintf (f, "%s ", msg);
			for (int i = 0; i < len; ++i)
			{
				fprintf (f, "%c", str[i]);
			}
			fprintf (f, "\n");
			fclose (f);
		}
		else
		{
			catxError (msg, str, len);
			exit (1);
		}
	}
}

void CatX::CatXBase::catxDebug (const char * msg, const int val) const
{
	if (strlen (DEBUG_FILE_NAME))
	{
		if (debugMode_)
		{
			FILE *f = fopen (DEBUG_FILE_NAME, "ab");
			fprintf (f, "%s %d\n", msg, val);
			fclose (f);
		}
		else
		{
			catxError (msg, val);
			exit (1);
		}

	}
}
