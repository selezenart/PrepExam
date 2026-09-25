#include <stdio.h>

char	*ft_strpbrk(const char *s1, const char *s2);

/* Print the offset of the match, not the pointer, so the output is stable. */
int	main(int argc, char **argv)
{
	char	*p;

	if (argc < 3)
		return (1);
	p = ft_strpbrk(argv[1], argv[2]);
	if (!p)
		printf("NULL\n");
	else
		printf("%ld [%s]\n", (long)(p - argv[1]), p);
	return (0);
}
