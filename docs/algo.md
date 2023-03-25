1. Get relative path of file
1. Create a directory in target + relative
1. Read the file
1. Find the first `let` keyword, saving everything before into a buffer
  - If we do not find `let`, error
1. Consume the last contiguous comment block from `let` (walk backwards)
1. The syntax for a declaration is:

```
// could be file comment

// documentation1
// doc2
let \s+ {identifier} \s+ = \s+ ({signature}) \s+ {
  {definition}
}
<EOF>
```