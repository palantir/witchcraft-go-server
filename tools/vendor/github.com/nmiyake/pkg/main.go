// +build module

// This file exists only to smooth the transition for modules. Having this file allows other modules to declare a
// dependency on the github.com/nmiyake/pkg module, which provides a mechanism for avoiding duplicate import path
// issues for modules that have a dependency on the pre-module github.com/nmiyake/pkg repository.
package main
