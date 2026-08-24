// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export const enSidebarConfig = sidebar({
  "/en/": [
    "",
    {
      text: "Guide",
      icon: "solar:book-2-linear",
      prefix: "guide/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "Architecture",
      icon: "solar:diagram-up-linear",
      prefix: "architecture/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "Deployment",
      icon: "solar:server-2-linear",
      prefix: "deployment/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "Development",
      icon: "solar:code-square-linear",
      prefix: "development/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
  ],
});
