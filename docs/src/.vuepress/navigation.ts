import { navbar, sidebar } from "vuepress-theme-hope";

type Locale = "zh" | "en";

const localePrefix: Record<Locale, string> = {
  zh: "/",
  en: "/en/",
};

const sections = [
  {
    path: "guide",
    icon: "solar:book-2-linear",
    label: { zh: "指南", en: "Guide" },
  },
  {
    path: "architecture",
    icon: "solar:diagram-up-linear",
    label: { zh: "架构", en: "Architecture" },
  },
  {
    path: "deployment",
    icon: "solar:server-2-linear",
    label: { zh: "部署", en: "Deployment" },
  },
  {
    path: "development",
    icon: "solar:code-square-linear",
    label: { zh: "开发", en: "Development" },
  },
] as const;

const githubItem = {
  text: "GitHub",
  icon: "mdi:github",
  link: "https://github.com/jzero-io/agentrazor",
};

const createNavbar = (locale: Locale) =>
  navbar([
    ...sections.map((section) => ({
      text: section.label[locale],
      icon: section.icon,
      link: `${localePrefix[locale]}${section.path}/`,
    })),
    githubItem,
  ]);

const createSidebar = (locale: Locale) => {
  const prefix = localePrefix[locale];

  return sidebar({
    [prefix]: [
      "",
      ...sections.map((section) => ({
        text: section.label[locale],
        icon: section.icon,
        prefix: `${section.path}/`,
        children: "structure" as const,
        collapsible: true,
        expanded: true,
      })),
    ],
  });
};

export const zhNavbar = createNavbar("zh");
export const enNavbar = createNavbar("en");
export const zhSidebar = createSidebar("zh");
export const enSidebar = createSidebar("en");
