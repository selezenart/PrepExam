#include <stdio.h>
#include <stdlib.h>

unsigned char	swap_bits(unsigned char octet);

int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	printf("%d\n", swap_bits((unsigned char)atoi(argv[1])));
	return (0);
}
