# Frontend dependencies

The application serves pinned assets locally. No CDN requests are needed at runtime.

| Asset | Version | Source | License |
| --- | --- | --- | --- |
| alpine.min.js | 3.17.5 | https://cdn.jsdelivr.net/npm/alpinejs@3.17.5/dist/cdn.min.js | MIT; see alpine.LICENSE.md |
| purify.min.js | 3.4.17 | https://cdn.jsdelivr.net/npm/dompurify@3.4.17/dist/purify.min.js | Apache-2.0 OR MPL-2.0; see purify.LICENSE and the source header |
| ui.css (generated utilities) | Tailwind CSS 3.4.17 | https://github.com/tailwindlabs/tailwindcss/tree/v3.4.17 | MIT; see tailwind.LICENSE |

To update a dependency, fetch an explicit version from its upstream package, keep
its license alongside the asset, and run the frontend tests and browser checks.
Regenerate `ui.css` using the command in `docs/development.md`.
