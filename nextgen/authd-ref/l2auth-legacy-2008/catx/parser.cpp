#include "parser.h"

namespace CatX
{
	Parser::Parser (const char * filename)
	{
		FILE *f = fopen (filename, "rb");
		if (f != NULL)
		{
			fseek (f, 0, SEEK_END);
			unsigned int size = ftell (f);
			fseek (f, 0, SEEK_SET);
			char * arr = (char *)malloc (sizeof (char) * size);
			fread (arr, sizeof (char), size, f);
			CatX::String fileData;
			bool isString = false;
			int isBlock = 0;
			for (unsigned int i = 0; i < size; ++i)
			{
				if (i & arr[i] == '"' && arr[i - 1] != '\\')
				{
					isString = !isString;
				}
				if ((arr[i] == '{' || arr[i] == '}') && !isString)
				{
					if (arr[i] == '{')
					{
						++isBlock;
					}
					else
					{
						--isBlock;
					}
				}
				if (!Func ().mycmp (arr, "//", i) || (arr[i] == ';' && !isBlock && !isString))
				{
					while (Func ().mycmp (arr, "\r\n", i) && ++i <= size);
				}
				else if (!Func ().mycmp (arr, "/*", i))
				{
					while (Func ().mycmp (arr, "*/", i) && ++i <= size);
					i += 2;
				}
				fileData += arr[i];
			}
			free (arr);
			CatX::String paramName;
			CatX::String paramData;
			isBlock = 0;
			isString = false;
			bool isParam = false;
			fileData += '\n';	// Dirty hack, I know
			for (unsigned int i = 0; i < fileData.length (); ++i)
			{
				if (!isString && fileData[i] == '=')
				{
					isParam = true;
					continue;
				}				
				else if (fileData[i] == '\n' && !isBlock && !isString)
				{
					if (paramName.length () && paramData.length ())
					{	
						data_[Func ().trim (paramName)] = Func().trim (paramData);
						paramName.clear ();
						paramData.clear ();
						isParam = false;
					}
				}
				if (!(isspace (fileData[i]) && !isString))
				{
					if (isParam)
					{
						paramData += fileData[i];
					}
					else
					{
						paramName += fileData[i];
					}
					if (!isString)
					{
						if (fileData[i] == '{')
						{
							++isBlock;					
						}
						else if (fileData[i] == '}')
						{
							--isBlock;
							if (!isBlock)
							{
								if (paramName.length () && paramData.length ())
								{
									data_[Func ().trim (paramName)] = Func().trim (paramData);
									paramName.clear ();
									paramData.clear ();
									isParam = false;
								}
							}
						}
					}
					if (i && fileData[i] == '"' && fileData[i - 1] != '\\')
					{
						isString = !isString;
					}
				}
			}
		}
		else
		{
			std::cerr << "File " << filename << " not found!" << std::endl;
			exit (1);
		}
	}

	int Parser::getInt (const CatX::String & param)
	{
		return CatX::String (data_[param]).toInt ();
	}

	double Parser::getDouble (const CatX::String & param)
	{
		return CatX::String (data_[param]).toDouble ();
	}

	std::vector <CatX::String> Parser::getVector (const CatX::String & str, const char razd)
	{
		std::vector <CatX::String> ret;
		bool isString = false;
		CatX::String data;
		int level = 0;
		for (unsigned int i = 1; i < str.length (); ++i)
		{
			if (i == str.length () - 1)
			{
				break;
			}
			if (str[i] == '{' && !isString)
			{
				++level;
			}
			else if (str[i] == '}' && !isString)
			{
				--level;
			}
			if (i && str[i] == '"' && str[i-1] != '\\')
			{
				isString = !isString;
			}
			if (str[i] == razd && !isString && !level)
			{
				if (data[0] == '"')
				{
					data = data.substr (1, data.length () - 2);
				}
				ret.push_back (data);
				data.clear ();
			}
			else
			{
				data += str[i];
			}
		}
		if (data[0] == '"')
		{
			data = data.substr (1, data.length () - 2);
		}
		ret.push_back (data);
		return ret;
	}

	CatX::String Parser::getString (const CatX::String & data)
	{
		return data_[data];
	}
}
