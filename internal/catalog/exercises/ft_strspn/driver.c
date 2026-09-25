#include <stdio.h>

size_t	ft_strspn(const char *s, const char *accept);

int	main(int argc, char **argv)
{
	if (argc < 3)
		return (1);
	printf("%zu\n", ft_strspn(argv[1], argv[2]));
	return (0);
}
