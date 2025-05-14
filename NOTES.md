
# Client C Code
A typical use of a Lemon parser might look something like the following:

    ParseTree *ParseFile(const char *zFilename){
        Tokenizer *pTokenizer;
        void *pParser;
        Token sToken;
        int hTokenId;
        ParserState sState;

        pTokenizer = TokenizerCreate(zFilename);
        pParser = ParseAlloc( malloc );
        InitParserState(&sState);
        while( GetNextToken(pTokenizer, &hTokenId, &sToken) ){
            Parse(pParser, hTokenId, sToken, &sState);
        }
        Parse(pParser, 0, sToken, &sState);
        ParseFree(pParser, free );
        TokenizerFree(pTokenizer);
        return sState.treeRoot;
    }

# References
1. [Lemon Overview](http://www.hwaci.com/sw/lemon/)
1. [The Lemon Parser Generator](https://sqlite.org/src/doc/trunk/doc/lemon.html)
1. [Understanding Lemon](http://www.gnudeveloper.com/groups/lemon-parser/understanding-lemon-generated-parser.html)
