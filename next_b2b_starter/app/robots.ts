import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: [
          "/api/",
          "/dashboard",
          "/dashboard/*",
          "/approvals",
          "/approvals/*",
        ],
      },
    ],
    sitemap: "https://yourdomain.com/sitemap.xml",
    host: "https://yourdomain.com",
  };
}
