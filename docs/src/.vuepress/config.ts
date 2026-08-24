// @ts-ignore
import { defineUserConfig } from "vuepress";
import theme from "./theme.js";

export default defineUserConfig({
  base: "/",

  locales: {
    "/": {
      lang: "zh-CN",
      title: "AgentRazor",
      description: "AgentRazor 文档",
    },
    "/en/": {
      lang: "en-US",
      title: "AgentRazor",
      description: "AgentRazor documentation",
    },
  },

  theme,
});
