# gomodtools
This contains various tools for working with go modules (currently just one
tool)

## gomodlayers
This command will take a set of go.mod files and report the dependencies
between them. It can be useful for establishing the order in which changes
and subsequent releases are made. It will report the layers of packages where
at each layer those packages only depend on the layers below.
