<!-- Created by mkdoc DO NOT EDIT. -->

# gomodlayers

This will take a list of go\.mod files \(or directories\) as trailing arguments
\(after &apos;\-\-&apos;\), parse them and print a report\. The report will show
how they relate to one another with regards to dependencies and can print them
in such an order that an earlier module does not depend on any subsequent
module\.

By default any report will be preceded with a description of what the various
columns mean\.

If one of the trailiing arguments does not end with &apos;/go\.mod&apos; then it
is taken as a directory name and the missing filename is automatically
appended\.



## Parameters

This uses the `param` package and so it has access to the help parameters
which give a comprehensive message describing the usage of the program and
the parameters you can give. The `-help` parameter on its own will print the
standard parameters that the program can accept but you can also give
parameters to show both more or less help, in more or less detail. Other
standard parameters allow you to explore where parameters have been set and
where they can be set. The description of the `-help` parameter is a good
place to start to explore the help available.

The intention of the `param` package is to provide complete documentation
for the program from the command line.


## Examples
For examples [see here](_gomodlayers.EXAMPLES.md)
