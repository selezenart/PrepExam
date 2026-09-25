#include <stdio.h>
#include <stdlib.h>
#include "ft_list_size.h"

int	ft_list_size(t_list *begin_list);

int	main(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	head = NULL;
	i = argc - 1;
	while (i >= 1)
	{
		node = malloc(sizeof(t_list));
		node->data = argv[i];
		node->next = head;
		head = node;
		i--;
	}
	printf("%d\n", ft_list_size(head));
	return (0);
}
