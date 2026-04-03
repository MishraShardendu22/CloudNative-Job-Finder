import path from "node:path";
import type { NextConfig } from "next";

const apiGatewayURL =
  process.env.API_GATEWAY_INTERNAL_URL ?? "http://api-gateway:8080";

const nextConfig: NextConfig = {
  reactCompiler: true,
  output: "standalone",
  turbopack: {
    root: path.join(__dirname, ".."),
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiGatewayURL}/:path*`,
      },
    ];
  },
};

export default nextConfig;
