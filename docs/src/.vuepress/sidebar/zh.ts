// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export const zhSidebarConfig = sidebar({
  "/": [
    "",
    {
      text: "使用指南",
      icon: "solar:book-2-linear",
      prefix: "guide/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "系统架构",
      icon: "solar:diagram-up-linear",
      prefix: "architecture/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "部署运维",
      icon: "solar:server-2-linear",
      prefix: "deployment/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
    {
      text: "开发指南",
      icon: "solar:code-square-linear",
      prefix: "development/",
      children: "structure",
      collapsible: true,
      expanded: true,
    },
  ],
});
