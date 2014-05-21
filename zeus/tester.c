#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "zeus.h"

int main() {
	char buf[256];
	char messagebuf[256];
	printf("%d %d\n", ZEUS(MyNodeId)(), ZEUS(NumberOfNodes)());
	fflush(stdout);
	while (fgets(buf, sizeof(buf), stdin) != NULL) {
		if (strlen(buf) > 0 && buf[strlen(buf)-1] == '\n')
			buf[strlen(buf)-1] = '\0';
		if (strlen(buf) == 0)
			continue;
		switch (buf[0]) {
			case 'R':
				{
					int source;
					if (buf[1] == '*')
						source = -1;
					else
						source = buf[1] - 'a';
					ZEUS(MessageInfo) mi = ZEUS(Receive)(source, messagebuf, sizeof(messagebuf) - 1);
					messagebuf[mi.length] = '\0';
					printf("%d %d %s\n", mi.sender_id, mi.length, messagebuf);
					fflush(stdout);
				}
				break;
			case 'S':
				{
					int dest = buf[1] - 'a';
					ZEUS(Send)(dest, buf + 2, strlen(buf + 2));
				}
				break;
			case 'Q':
				{
					int code = buf[1] - '0';
					exit(code);
				}
				break;
			default:
				assert(0);
		}
	}
	return 0;
}
