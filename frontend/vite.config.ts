import { resolve } from "node:path";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite-plus";

// make gen writes this file in its own format, which is not the format of the formatter. Skip the file.
const generatedFile = "src/api/schema.d.ts";

function readPort(env: Record<string, string>, name: string, fallback: number): number {
  const value = env[name];
  if (value === undefined || value === "") {
    return fallback;
  }
  const port = Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`${name} is not valid. Use a number from 1 to 65535.`);
  }
  return port;
}

export default defineConfig(({ mode }) => {
  // The settings come from the .env file in the root of the repo, and from the environment.
  // Read only the 2 variables that this file uses.
  const env = loadEnv(mode, resolve(import.meta.dirname, ".."), ["WEB_PORT", "API_PORT"]);
  const apiTarget = `http://localhost:${readPort(env, "API_PORT", 5653)}`;

  return {
    plugins: [react(), tailwindcss()],
    // The settings VITE_* of the page come from the .env file in the root of the repo too.
    envDir: resolve(import.meta.dirname, ".."),
    server: {
      port: readPort(env, "WEB_PORT", 5651),
      strictPort: true,
      proxy: {
        "/api": apiTarget,
        "/docs": apiTarget,
        "/openapi.yaml": apiTarget,
      },
    },
    test: {
      environment: "jsdom",
      setupFiles: ["./test/setup.ts"],
      include: ["test/unit/**/*.test.{ts,tsx}"],
    },
    fmt: { ignorePatterns: [generatedFile] },
    lint: {
      ignorePatterns: [generatedFile],
      jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
      rules: { "vite-plus/prefer-vite-plus-imports": "error" },
      options: { typeAware: true, typeCheck: true },
    },
  };
});
