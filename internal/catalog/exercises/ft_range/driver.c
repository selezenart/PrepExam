#include <stdio.h>
#include <stdlib.h>

int	*ft_range(int start, int end);

int	main(int argc, char **argv)
{
	int	start;
	int	end;
	int	len;
	int	i;
	int	*range;

	if (argc < 3)
		return (1);
	start = atoi(argv[1]);
	end = atoi(argv[2]);
	range = ft_range(start, end);
	if (!range)
	{
		printf("(null)\n");
		return (0);
	}
	if (start > end)
		len = start - end + 1;
	else
		len = end - start + 1;
	i = 0;
	while (i < len)
	{
		printf("%d ", range[i]);
		i++;
	}
	printf("|end\n");
	return (0);
}
