import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const outDir = resolve("docs");
await mkdir(outDir, { recursive: true });

const html = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>DADdy Documentation</title>
    <style>
      :root {
        --bg: #f8fafc;
        --ink: #0f172a;
        --muted: #334155;
        --card: #ffffff;
        --brand: #0f766e;
        --brand-2: #14b8a6;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif;
        color: var(--ink);
        background:
          radial-gradient(circle at 10% -5%, #ccfbf1 0%, transparent 45%),
          radial-gradient(circle at 110% 5%, #bae6fd 0%, transparent 40%),
          var(--bg);
      }
      .wrap {
        max-width: 980px;
        margin: 0 auto;
        padding: 56px 20px;
      }
      h1 {
        margin: 0 0 10px;
        font-size: clamp(2rem, 4vw, 3rem);
        letter-spacing: -0.02em;
      }
      p { color: var(--muted); margin: 0 0 28px; }
      .grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
        gap: 16px;
      }
      .card {
        display: block;
        background: var(--card);
        border: 1px solid #e2e8f0;
        border-radius: 16px;
        padding: 20px;
        text-decoration: none;
        color: inherit;
        box-shadow: 0 6px 24px rgba(2, 6, 23, 0.08);
        transition: transform 0.15s ease, box-shadow 0.15s ease;
      }
      .card:hover {
        transform: translateY(-3px);
        box-shadow: 0 12px 30px rgba(2, 6, 23, 0.12);
      }
      .badge {
        display: inline-block;
        font-size: 12px;
        font-weight: 700;
        background: linear-gradient(90deg, var(--brand), var(--brand-2));
        color: white;
        border-radius: 999px;
        padding: 5px 10px;
        margin-bottom: 10px;
      }
      .title { font-size: 1.2rem; font-weight: 800; margin: 0 0 6px; }
      .desc { margin: 0; color: var(--muted); }
    </style>
  </head>
  <body>
    <main class="wrap">
      <h1>DADdy Docs</h1>
      <p>Auto-generated documentation for backend APIs and frontend source.</p>
      <section class="grid">
        <a class="card" href="./api/index.html">
          <span class="badge">OpenAPI</span>
          <h2 class="title">API Documentation</h2>
          <p class="desc">Interactive API reference generated from the OpenAPI spec.</p>
        </a>
        <a class="card" href="./web/index.html">
          <span class="badge">TypeDoc</span>
          <h2 class="title">Frontend Documentation</h2>
          <p class="desc">Frontend symbols and modules generated from TypeScript sources.</p>
        </a>
      </section>
    </main>
  </body>
</html>
`;

await writeFile(resolve(outDir, "index.html"), html, "utf8");
