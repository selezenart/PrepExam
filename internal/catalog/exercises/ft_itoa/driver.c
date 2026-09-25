#include <stdio.h>
#include <stdlib.h>

char	*ft_itoa(int nbr);

int	main(int argc, char **argv)
{
	char	*s;

	if (argc < 2)
		return (1);
	s = ft_itoa(atoi(argv[1]));
	if (!s)
		printf("(null)\n");
	else
		printf("[%s]\n", s);
	return (0);
}
