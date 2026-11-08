This is github.com/lib-x/mdx v0.1.15, copied under its original license. The
key-block-info decoder handles the uncompressed metadata used by MDX versions
below 2.0. `legacy_test.go` covers that compatibility fix. `mdict_v3.go` reads
engine 3.0 dictionaries (MDXBuilder 4.x): typed key/record blocks, UTF-8 keys
for both MDX and MDD, and per-block fast-XOR or Salsa20/8 encryption.
`mdict_v3_test.go` covers that reader. `splitKeyBlock` bounds-checks UTF-16
terminators and truncated headers so corrupt key blocks fail closed instead of
panicking (`mdict_split_key_test.go`). The application uses this local copy
through the `replace` directive in the root go.mod.
