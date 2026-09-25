#include <stdio.h>
#include <stdlib.h>
#include "ft_list.h"

void	ft_list_foreach(t_list *begin_list, void (*f)(void *));

void	print_data(void *data)
{
	printf("[%s]\n", (char *)data);
}

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
	ft_list_foreach(head, &print_data);
	printf("|end\n");
	return (0);
}
