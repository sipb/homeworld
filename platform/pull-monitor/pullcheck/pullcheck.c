#include <stdio.h>

int main(int argc, char *argv[]) {
	if (argc != 2) {
		return 1;
	}
	printf("hello container world [%s]\n", argv[1]);
	return 0;
}
