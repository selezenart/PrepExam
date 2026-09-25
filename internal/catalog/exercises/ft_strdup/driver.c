#include <stdio.h>

char	*ft_strdup(char *src);

int	main(int argc, char **argv)
{
	char	*copy;

	if (argc < 2)
		return (1);
	copy = ft_strdup(argv[1]);
	if (!copy)
	{
		printf("(null)\n");
		return (0);
	}
	printf("[%s]\n", copy);
	printf("is_a_copy=%d\n", copy != argv[1]);
	return (0);
}
