#include <stdio.h>
#include <stdlib.h>

unsigned int	lcm(unsigned int a, unsigned int b);

int	main(int argc, char **argv)
{
	if (argc < 3)
		return (1);
	printf("%u\n", lcm((unsigned int)strtoul(argv[1], NULL, 10),
			(unsigned int)strtoul(argv[2], NULL, 10)));
	return (0);
}
