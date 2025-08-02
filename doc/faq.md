# FAQ
## How can I show/hide columns
On the right hand side of the column names there are 3 dots, which open a menu to toggle the visible columns.

## I forgot my admin password, how can I reset it?

Run this command to update your password
```
OPENSOHO_SHARED_SECRET=x ./opensoho superuser update <email> <new password>
```

## Can I reorder the collections?

No that's currently not possible (this is a limitations of pocketbase).
But collections can be *pinned* by clicking the little pushpin next to the collection name in the collection list.
