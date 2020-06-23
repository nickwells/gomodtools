<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
gomodlayers -names-by-level -- dir1/go.mod dir2/go.mod dir3/go.mod
```
This will print just the names of the modules but in an order such that no
module depends on any of the modules listed after it. This can be useful when
you want to know the best order to update the modules.

```sh
gomodlayers -- dir1/go.mod dir2/go.mod dir3/go.mod
```
This will print the default output: an extensive introduction explaining the
results, column headings and then the modules in an order such that no module
depends on any of the modules listed after it. The columns shown are the module
level, the full module name and how many of the other modules use that module.

