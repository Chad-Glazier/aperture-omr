/*
This package provides an interface to collect information about the system in
which the OMR is running. It passively checks certain statistics and caches
them in memory and on disk to provide stable access, tracks certain configured
constants (i.e., user- and runtime-determined resource limits), and exposes
logging functions that can be used globally.
*/
package sys
