import { hopeTheme } from "vuepress-theme-hope";
import {
  enNavbar,
  enSidebar,
  zhNavbar,
  zhSidebar,
} from "./navigation.js";

export default hopeTheme({
  hostname: "https://docs.agentrazor.jzero.io",
  author: {
    name: "jaronnie",
    url: "https://github.com/jaronnie",
  },
  copyright: "Copyright © 2026 jzero-io",
  logo: "/favicon.ico",
  repo: "jzero-io/agentrazor",
  docsDir: "docs/src",
  locales: {
    "/": {
      navbar: zhNavbar,
      sidebar: zhSidebar,
      footer: "",
      displayFooter: true,
      metaLocales: {
        editLink: "在 GitHub 上编辑此页",
      },
    },
    "/en/": {
      navbar: enNavbar,
      sidebar: enSidebar,
      metaLocales: {
        editLink: "Edit this page on GitHub",
      },
    },
  },
  markdown: {
    codeTabs: true,
    gfm: true,
    imgLazyload: true,
    imgSize: true,
    mermaid: true,
  },
  plugins: {
    icon: {
      assets: "iconify",
    },
  },
});
