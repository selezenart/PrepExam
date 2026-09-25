#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "list.h"

t_list	*sort_list(t_list *lst, int (*cmp)(int, int));

int	ascending(int a, int b)
{
	return (a <= b);
}

int	descending(int a, int b)
{
	return (a >= b);
}

/* argv[1] is "asc" or "desc"; the list is built from the rest. */
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
		node->data = atoi(argv[i]);
		node->next = head;
		head = node;
		i--;
	}
	if (strcmp(argv[1], "desc") == 0)
		head = sort_list(head, &descending);
	else
		head = sort_list(head, &ascending);
	node = head;
	while (node)
	{
		printf("%d ", node->data);
		node = node->next;
	}
	printf("|end\n");
	return (0);
}
