package parser

//go:generate cp ../tool/lempar.c lempar.c.template

// This file is needed to make Go ignore the lempar.c file
// when building the package while still allowing us to embed it.
