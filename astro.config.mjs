import { defineConfig } from "astro/config";
import mdx from "@astrojs/mdx";
import { unified } from "@astrojs/markdown-remark";
import rehypeMermaid from "rehype-mermaid";
import { visit } from "unist-util-visit";

function externalLinksInNewTabs() {
  return (tree) => {
    visit(tree, "element", (node) => {
      if (node.tagName !== "a") {
        return;
      }

      const href = String(node.properties?.href ?? "");

      if (href.startsWith("https://") || href.startsWith("http://")) {
        node.properties.target = "_blank";
        node.properties.rel = "noopener noreferrer";
      }
    });
  };
}

export default defineConfig({
  build: {
    inlineStylesheets: "never",
  },
  integrations: [mdx()],
  markdown: {
    syntaxHighlight: {
      type: "shiki",
      excludeLangs: ["mermaid"],
    },
    shikiConfig: {
      theme: "one-dark-pro",
    },
    processor: unified({
      rehypePlugins: [
        externalLinksInNewTabs,
        [
          rehypeMermaid,
          {
            strategy: "img-svg",
            colorScheme: "dark",
            mermaidConfig: {
              theme: "dark",
            },
          },
        ],
      ],
    }),
  },
  output: "static",
});
