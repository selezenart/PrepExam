#include <stdio.h>

char	*ft_strrev(char *str);

int	main(int argc, char **argv)
{
	char	*r;

	if (argc < 2)
		return (1);
	r = ft_strrev(argv[1]);
	printf("[%s]\n", argv[1]);
	printf("returns_str=%d\n", r == argv[1]);
	return (0);
}
