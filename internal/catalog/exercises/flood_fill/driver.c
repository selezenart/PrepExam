#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "flood_fill.h"

/* argv[1] and argv[2] are begin.x and begin.y; every later argument is one
   row of the grid, all the same width. */
int	main(int argc, char **argv)
{
	t_point	size;
	t_point	begin;
	char	**tab;
	int		i;

	if (argc < 4)
		return (1);
	begin.x = atoi(argv[1]);
	begin.y = atoi(argv[2]);
	size.y = argc - 3;
	size.x = (int)strlen(argv[3]);
	tab = malloc(sizeof(char *) * size.y);
	i = 0;
	while (i < size.y)
	{
		tab[i] = strdup(argv[i + 3]);
		i++;
	}
	flood_fill(tab, size, begin);
	i = 0;
	while (i < size.y)
	{
		printf("%s\n", tab[i]);
		i++;
	}
	return (0);
}
