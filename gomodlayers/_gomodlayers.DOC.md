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





## Examples
For examples [see here](_gomodlayers.EXAMPLES.md)
