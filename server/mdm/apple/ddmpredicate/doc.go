// Package ddmpredicate parses and validates predicate strings used by Apple
// Declarative Device Management (DDM) activations.
//
// The grammar follows Apple's published BNF for NSPredicate format strings:
// https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/Predicates/Articles/pBNF.html
//
// DDM predicates reference device state through @-prefixed key path
// functions. This package only accepts the three functions Apple defines,
// and each must be called with a parenthesized key path:
//
//	@status(device.model.family)   status report items
//	@property(age)                 server-set management properties
//	@key(identifier)               keys of objects inside a collection,
//	                               typically via a SUBQUERY variable
//
// Bare references such as @status without parentheses, and unknown
// functions such as @invalid, are rejected. Standard NSPredicate
// collection operators (@count, @avg, @sum, ...) remain valid because
// Apple's own DDM examples use SUBQUERY(...).@count.
//
// The archive BNF predates some syntax that Apple's DDM examples and the
// Predicate Format String Syntax guide rely on, so the parser additionally
// accepts:
//
//   - the operator spellings ==, <>, =>, and =< as synonyms for =, !=, >=,
//     and <=
//   - && , || , and ! as synonyms for AND, OR, and NOT
//   - SUBQUERY(collection, $variable, predicate)
//   - string comparison options such as [cd] on the equality and ordering
//     operators, not only on the string operators
//
// Keywords (AND, IN, TRUEPREDICATE, true, ...) are case-insensitive, as in
// NSPredicate. Key path function names (@status), collection operators
// (@count, @firstObject) and BNF function names (sum, now, ...) are
// case-sensitive.
//
// Format arguments (%@, %K, ...) are rejected: DDM predicates are shipped
// to devices as literal strings and evaluated without substitution
// arguments.
package ddmpredicate
