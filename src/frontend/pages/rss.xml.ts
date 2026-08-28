import type { APIRoute } from "astro";
import { getCollection } from "astro:content";

const escapeXML = (value: string) =>
  value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");

export const GET: APIRoute = async ({ site }) => {
  const posts = (await getCollection("blog")).sort((a, b) =>
    b.data.date.localeCompare(a.data.date),
  );
  const items = posts.map((post) => {
    const url = new URL(`/posts/${post.id}/`, site).href;
    const published = new Date(`${post.data.date}T00:00:00Z`).toUTCString();

    return `<item><title>${escapeXML(post.data.title)}</title><link>${url}</link><guid isPermaLink="true">${url}</guid><pubDate>${published}</pubDate><description>${escapeXML(post.data.summary)}</description></item>`;
  });

  return new Response(
    `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Mateusz Rembiasz Blog</title><link>${new URL("/", site).href}</link><description>Technical articles about AWS, infrastructure, CI/CD, and serverless systems.</description><language>en</language>${items.join("")}</channel></rss>`,
    { headers: { "Content-Type": "application/rss+xml; charset=utf-8" } },
  );
};
