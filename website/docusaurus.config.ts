import type * as Preset from "@docusaurus/preset-classic";
import type { Config } from "@docusaurus/types";
import { themes as prismThemes } from "prism-react-renderer";

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  title: "Bison",
  tagline: "Kubernetes-based GPU Resource Billing and Scheduling Platform",
  favicon: "img/favicon.ico",

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // Set the production url of your site here
  url: "https://bison.lei6393.com",
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: "/",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "SuperMarioYL", // Usually your GitHub org/user name.
  projectName: "Bison", // Usually your repo name.

  onBrokenLinks: "warn",

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en", "zh-Hans"],
  },

  markdown: {
    mermaid: true,
  },

  themes: ["@docusaurus/theme-mermaid"],

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: "./sidebars.ts",
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl: "https://github.com/SuperMarioYL/Bison/tree/main/website/",
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
        },
        blog: false, // Disable blog for now
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    image: "img/bison-social-card.png",
    metadata: [
      { name: 'google-site-verification', content: 'AKwgEpVubW0iVPeZN2RkopUMJbx4K-Yy8aWwCZq-T34' },
      { name: 'baidu-site-verification', content: 'codeva-E3148CbKk3' },
    ],
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: "Bison",
      logo: {
        alt: "Bison Logo",
        src: "img/logo.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "tutorialSidebar",
          position: "left",
          label: "Documentation",
        },
        {
          type: "docsVersionDropdown",
          position: "right",
        },
        {
          type: "localeDropdown",
          position: "right",
        },
        {
          href: "https://github.com/SuperMarioYL/Bison",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Docs",
          items: [
            {
              label: "Getting Started",
              to: "/docs",
            },
            {
              label: "Architecture",
              to: "/docs/architecture",
            },
          ],
        },
        {
          title: "Resources",
          items: [
            {
              label: "Helm Charts",
              href: "https://supermarioyl.github.io/Bison/charts/",
            },
            {
              label: "GitHub Releases",
              href: "https://github.com/SuperMarioYL/Bison/releases",
            },
          ],
        },
        {
          title: "More",
          items: [
            {
              label: "GitHub",
              href: "https://github.com/SuperMarioYL/Bison",
            },
            {
              label: "Issues",
              href: "https://github.com/SuperMarioYL/Bison/issues",
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Bison Project. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
