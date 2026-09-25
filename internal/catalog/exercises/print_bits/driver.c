#include <stdlib.h>
#include <unistd.h>

void	print_bits(unsigned char octet);

int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	print_bits((unsigned char)atoi(argv[1]));
	write(1, "|end\n", 5);
	return (0);
}
