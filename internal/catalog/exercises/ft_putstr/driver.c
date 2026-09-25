#include <unistd.h>

void	ft_putstr(char *str);

int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	ft_putstr(argv[1]);
	write(1, "|end\n", 5);
	return (0);
}
