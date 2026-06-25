import { cp, mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import swaggerUiDist from "swagger-ui-dist";

const outDir = resolve("docs/api");
const specSource = resolve("apps/breezy-api/spec/openapi.yaml");
const specTarget = resolve(outDir, "openapi.yaml");

const distPath =
  (typeof swaggerUiDist.getAbsoluteFSPath === "function" && swaggerUiDist.getAbsoluteFSPath()) ||
  (typeof swaggerUiDist.absolutePath === "function" && swaggerUiDist.absolutePath());

if (!distPath) {
  throw new Error("Unable to locate swagger-ui-dist assets.");
}

await mkdir(outDir, { recursive: true });
await cp(distPath, outDir, { recursive: true });

const rawSpec = await readFile(specSource, "utf8");
await writeFile(specTarget, rawSpec, "utf8");

const html = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>DADdy API Docs</title>
    <link rel="stylesheet" href="./swagger-ui.css" />
    <style>
      html {
        box-sizing: border-box;
        overflow-y: scroll;
      }
      *, *:before, *:after {
        box-sizing: inherit;
      }
      body {
        margin: 0;
        background: linear-gradient(180deg, #ecfeff 0%, #ffffff 42%);
      }
      .topbar {
        display: none;
      }
      .swagger-ui .scheme-container {
        background: #f0fdfa;
        border: 1px solid #99f6e4;
      }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="./swagger-ui-bundle.js"></script>
    <script src="./swagger-ui-standalone-preset.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: "./openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "BaseLayout",
        displayRequestDuration: true,
      });
    </script>
  </body>
</html>
`;

await writeFile(resolve(outDir, "index.html"), html, "utf8");
