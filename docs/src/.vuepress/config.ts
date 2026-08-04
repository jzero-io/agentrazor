// @ts-ignore
import { defineUserConfig } from "vuepress";
import theme from "./theme.js";

export default defineUserConfig({
  base: "/",

  locales: {
    "/": {
      lang: "zh-CN",
      title: "agentrazor-admin",
      description: "agentrazor-admin 文档",
    },
    "/en/": {
      lang: "en-US",
      title: "agentrazor-admin",
      description: "docs for agentrazor-admin",
    },
  },

  theme,

  // 和 PWA 一起启用
  // shouldPrefetch: false,
});
