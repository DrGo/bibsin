package bibsin

// Package bibsin parses bibtex files and perform sorting and deduplication operations
// on bibilio records.

// BNF
// Database     ::= (Junk '@' Entry)*
// Entry        ::= Record
//               |  Comment
//               |  String
//               |  Preamble
// Comment      ::= "comment" [^\n]* \n                -- ignored
// String       ::= "string" '{' Field* '}'             -- not handled 
// Preamble     ::= "preamble" '{' .* '}'         -- not handled
// Record       ::= Type '{' Key ',' Field* '}'
//               |  Type '(' Key ',' Field* ')' -- not handled
// Type         ::= Name
// Key          ::= Name
// Field        ::= Name '=' Value
// Name         ::= [^\s\"#%'(){}]*
// Value        ::= [0-9]+						-- not handled
//               |  '"' ([^'"']|\\'"')* '"'  -- not handled 
//               |  '{' .* '}'                         -- (balanced)

