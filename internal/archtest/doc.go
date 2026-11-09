// Package archtest holds repository-structure tests: the places where a
// package is allowed to reach another, and the writes that must go through
// an atomic helper. Its tests read source files; nothing imports it.
package archtest
