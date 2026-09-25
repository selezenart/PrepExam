#include <stdio.h>
#include <stdlib.h>

void	sort_int_tab(int *tab, unsigned int size);

int	main(int argc, char **argv)
{
	int		tab[256];
	int		n;
	int		i;

	n = 0;
	while (n < argc - 1 && n < 256)
	{
		tab[n] = atoi(argv[n + 1]);
		n++;
	}
	sort_int_tab(tab, (unsigned int)n);
	i = 0;
	while (i < n)
	{
		printf("%d ", tab[i]);
		i++;
	}
	printf("|end\n");
	return (0);
}
