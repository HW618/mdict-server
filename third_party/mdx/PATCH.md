This is github.com/lib-x/mdx v0.1.15, copied under its original license. The
key-block-info decoder handles the uncompressed metadata used by MDX versions
below 2.0. `legacy_test.go` covers that compatibility fix. The application
uses this local copy through the `replace` directive in the root go.mod.
