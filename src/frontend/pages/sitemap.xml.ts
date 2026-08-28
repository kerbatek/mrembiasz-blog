import type { APIRoute } from "astro";
import { getCollection } from "astro:content";

export const GET: APIRoute = async ({ site }) => {
  const posts = await getCollection("blog");
  const urls = [
    `<url><loc>${new URL("/", site).href}</loc></url>`,
    ...posts.map((post) => {
      const url = new URL(`/posts/${post.id}/`, site).href;
      return `<url><loc>${url}</loc><lastmod>${post.data.date}</lastmod></url>`;
    }),
  ];

  return new Response(
    `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join("")}</urlset>`,
    { headers: { "Content-Type": "application/xml; charset=utf-8" } },
  );
};
