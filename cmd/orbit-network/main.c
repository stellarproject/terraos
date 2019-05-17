#define _GNU_SOURCE

/*
	Copyright (c) 2019 Stellar Project

	Permission is hereby granted, free of charge, to any person
	obtaining a copy of this software and associated documentation
	files (the "Software"), to deal in the Software without
	restriction, including without limitation the rights to use, copy,
	modify, merge, publish, distribute, sublicense, and/or sell copies
	of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED,
	INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
	HOLDERS BE LIABLE FOR ANY CLAIM,
	DAMAGES OR OTHER LIABILITY,
	WHETHER IN AN ACTION OF CONTRACT,
	TORT OR OTHERWISE,
	ARISING FROM, OUT OF OR IN CONNECTION WITH
	THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#include <stdio.h>
#include <stdlib.h>
#include <sched.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/types.h>

int main(int argc, char **argv)
{
	if (argc != 2) {
		fprintf(stderr, "no path specified\n");
	}
	char *path = argv[1];
	int f = open(path, O_RDWR | O_CREAT);
	if (f < 0) {
		return 1;
	}
	close(f);
	if (unshare(CLONE_NEWNET) != 0) {
		fprintf(stderr, "%s: clone failed", strerror(errno));
		return 1;
	}
	if (mount("/proc/self/ns/net", path, "none", MS_BIND, NULL) != 0) {
		fprintf(stderr, "%s: clone failed", strerror(errno));
		return 1;
	}
	return 0;
}
