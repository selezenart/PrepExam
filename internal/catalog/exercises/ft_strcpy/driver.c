#include <stdio.h>
#include <string.h>

char	*ft_strcpy(char *s1, char *s2);

int	main(int argc, char **argv)
{
	char	dest[1024];
	char	*ret;

	if (argc < 2)
		return (1);
	memset(dest, '#', sizeof(dest));
	ret = ft_strcpy(dest, argv[1]);
	printf("[%s]\n", dest);
	printf("returns_dest=%d\n", ret == dest);
	printf("byte_after_nul=%c\n", dest[strlen(argv[1]) + 1]);
	return (0);
}
