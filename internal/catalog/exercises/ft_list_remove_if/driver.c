#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "ft_list.h"

void	ft_list_remove_if(t_list **begin_list, void *data_ref, int (*cmp)());

int	cmp_str(void *a, void *b)
{
	return (strcmp((char *)a, (char *)b));
}

/* argv[1] is the data to remove; the list is built from the rest. */
int	main(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	if (argc < 2)
		return (1);
	head = NULL;
	i = argc - 1;
	while (i >= 2)
	{
		node = malloc(sizeof(t_list));
		node->data = argv[i];
		node->next = head;
		head = node;
		i--;
	}
	ft_list_remove_if(&head, argv[1], &cmp_str);
	node = head;
	while (node)
	{
		printf("[%s]\n", (char *)node->data);
		node = node->next;
	}
	printf("|end\n");
	return (0);
}
