// vitest/config re-exports Vite's defineConfig with the `test` key typed.
import { defineConfig, type Plugin } from 'vitest/config';

// The site is published to https://<user>.github.io/wordle-helper/, so assets
// have to be resolved relative to that sub-path rather than the domain root.
// Overridable for other hosts: BASE_PATH=/ npm run build
const base = process.env['BASE_PATH'] ?? '/wordle-helper/';

// The page is entirely self-contained: no fonts, no analytics, no API. Saying
// so in a policy means a future dependency cannot quietly add a request.
// `connect-src 'none'` blocks fetch, XHR, WebSockets and beacons outright.
//
// `frame-ancestors` is deliberately absent: browsers ignore it when it arrives
// via <meta> and log an error for it, and GitHub Pages cannot set response
// headers, so there is nowhere to put it that would take effect.
const CONTENT_SECURITY_POLICY = [
  "default-src 'none'",
  "script-src 'self'",
  "style-src 'self'",
  "img-src 'self' data:",
  "worker-src 'self'",
  "connect-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
].join('; ');

/**
 * Injects the policy into the built page only.
 *
 * Vite's dev server needs inline styles and a websocket for hot reload, so
 * applying this during development would break the thing it is protecting.
 */
function contentSecurityPolicy(): Plugin {
  return {
    name: 'wordle-helper:csp',
    apply: 'build',
    transformIndexHtml() {
      return [
        {
          tag: 'meta',
          attrs: { 'http-equiv': 'Content-Security-Policy', content: CONTENT_SECURITY_POLICY },
          injectTo: 'head-prepend',
        },
      ];
    },
  };
}

export default defineConfig({
  base,
  plugins: [contentSecurityPolicy()],
  build: {
    target: 'es2022',
    // The word list dwarfs everything else and is pure data, so warning about
    // the worker chunk's size would only be noise.
    chunkSizeWarningLimit: 700,
  },
  worker: {
    format: 'es',
  },
  test: {
    environment: 'node',
    include: ['test/**/*.test.ts'],
  },
});
