# Synthetic MDX fixtures

These tiny UTF-8, MDX 2.0 files contain only self-authored `<p>word</p>` definitions:

- `primary.mdx`: `Apple`, `apple`, `apricot`, `Zebra`, `中文`, `中文词典`.
- `secondary.mdx`: `apple`, `application`.

They were generated with `mdict_utils.base.writemdict.MDictWriter`, with the
filename stem as the title and `Synthetic regression fixture; no third-party
dictionary content.` as the description. Normal tests read the committed files
and require no Python dependencies. The fixtures exercise MDX loading, duplicate
words across dictionaries, case normalization, Unicode, filtering and reloads.
