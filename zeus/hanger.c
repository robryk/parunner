// A binary that hangs in the middle of receiving a message
#include <assert.h>
#include <stdio.h>

const char command[] = {
	0x04, // receive
	0xff, // source = int32(-1) little endian
	0xff,
	0xff,
	0xff,
};

int main() {
	char buf[20];
	FILE* cmdin = fdopen(3, "r");
	assert(cmdin);
	FILE* cmdout = fdopen(4, "w");
	assert(cmdout);
	assert(fwrite(command, sizeof(command), 1, cmdout) == 1);
	assert(fflush(cmdout) == 0);
	assert(fread(buf, 1, sizeof(buf), cmdin) >= 0);
	return 0;
}
	

