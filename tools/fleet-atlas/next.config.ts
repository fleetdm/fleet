import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: true,
  // Allow dev-server requests coming through cloudflared / ngrok tunnels so
  // teammates can view client components (React Flow, quizzes, etc.) without
  // Next's dev-mode cross-origin blocker intervening.
  allowedDevOrigins: ["*.trycloudflare.com", "*.ngrok.app", "*.ngrok-free.app"],
  turbopack: {
    // Pin Turbopack's workspace root to this directory. Without this, when
    // Atlas lives at fleetdm/fleet/tools/fleet-atlas, Turbopack walks up and
    // picks up Fleet's root postcss.config.js and tsconfig.json, which
    // expect deps (autoprefixer, @tsconfig/recommended) that don't exist in
    // Atlas's node_modules.
    root: import.meta.dirname,
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
