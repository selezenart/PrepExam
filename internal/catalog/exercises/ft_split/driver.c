#include <stdio.h>

char	**ft_split(char *str);

int	main(int argc, char **argv)
{
	char	**words;
	int		i;

	if (argc < 2)
		return (1);
	words = ft_split(argv[1]);
	if (!words)
	{
		printf("(null)\n");
		return (0);
	}
	i = 0;
	while (words[i])
	{
		printf("[%s]\n", words[i]);
		i++;
	}
	printf("count=%d\n", i);
	return (0);
}
